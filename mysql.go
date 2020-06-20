package runc_mysql

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mholt/archiver/v3"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/urfave/cli"
)

const BundleFileUri = "https://public-1251825869.cos.ap-shanghai.myqcloud.com/alpine.tar.gz"

type RuncMysql struct {
	ep            string
	rootPassword  string
	dir           string
	bundleZipFile string
	name          string
}

func New(ep, rootPassword string) *RuncMysql {
	c := &RuncMysql{
		ep:            ep,
		rootPassword:  rootPassword,
		bundleZipFile: "mysql.tar.gz",
		name:          "mysql",
	}

	return c
}

func (c *RuncMysql) Load() error {
	dir, err := ioutil.TempDir(os.TempDir(), "mysql*")
	if err != nil {
		return err
	}
	c.dir = dir
	bundleFilePath := filepath.Join(dir, c.bundleZipFile)
	err = downLoadFile(BundleFileUri, bundleFilePath)
	if err != nil {
		return err
	}

	os.Chdir(c.dir)
	err = extractTarGz(bundleFilePath)
	if err != nil {
		return err
	}
	return nil
}

func MockFlagSet(args []string) *flag.FlagSet {
	f := flag.NewFlagSet("", 0)
	err := f.Parse(args)
	if err != nil {
		log.Fatal(err)
	}
	return f
}

func (c *RuncMysql) Start() error {
	os.Chdir(c.dir)
	context := cli.NewContext(cli.NewApp(), MockFlagSet([]string{"t", c.name}), nil)
	if err := checkArgs(context, 1, exactArgs); err != nil {
		return err
	}
	if err := revisePidFile(context); err != nil {
		return err
	}
	spec, err := setupSpec(context)
	if err != nil {
		return err
	}
	_, err = startContainer(context, spec, CT_ACT_RUN, nil)
	return err
}

func (c *RuncMysql) ShutDown(force bool) error {
	context := cli.NewContext(cli.NewApp(), MockFlagSet([]string{"t", c.name}), nil)
	if err := checkArgs(context, 1, exactArgs); err != nil {
		return err
	}

	id := context.Args().First()
	container, err := getContainer(context)
	if err != nil {
		if lerr, ok := err.(libcontainer.Error); ok && lerr.Code() == libcontainer.ContainerNotExists {
			// if there was an aborted start or something of the sort then the container's directory could exist but
			// libcontainer does not see it because the state.json file inside that directory was never created.
			path := filepath.Join(context.GlobalString("root"), id)
			if e := os.RemoveAll(path); e != nil {
				fmt.Fprintf(os.Stderr, "remove %s: %v\n", path, e)
			}
			if force {
				return nil
			}
		}
		return err
	}
	s, err := container.Status()
	if err != nil {
		return err
	}
	switch s {
	case libcontainer.Stopped:
		destroy(container)
	case libcontainer.Created:
		return killContainer(container)
	default:
		if force {
			return killContainer(container)
		}
		return fmt.Errorf("cannot delete container %s that is not stopped: %s\n", id, s)
	}
	return nil
}

func downLoadFile(src, dest string) error {
	sqlBundle, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create bundleFile failed, %w", err)
	}
	defer sqlBundle.Close()
	resp, err := http.Get(src)
	if err != nil {
		return fmt.Errorf("download bundleFile failed, %w", err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(sqlBundle, resp.Body)
	return err
}

func extractTarGz(bundleFilePath string) error {
	err := archiver.Unarchive(bundleFilePath, ".")
	return err
}
