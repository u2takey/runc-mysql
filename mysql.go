package runc_mysql

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/urfave/cli"
)

const DefaultBundleFileUri = "https://public-1251825869.cos.ap-shanghai.myqcloud.com/mysql.tar.gz"

type RuncMysql struct {
	initDatabase  string
	rootPassword  string
	dir           string
	bundleZipFile string
	name          string
	bundleFileUri string
}

func New(bundleFileUri ...string) *RuncMysql {
	c := &RuncMysql{
		rootPassword:  "root",
		initDatabase:  "test",
		bundleZipFile: "mysql.tar.gz",
		name:          "mysql",
		dir:           filepath.Join(os.TempDir(), "mysql"),
	}
	if len(bundleFileUri) > 0 {
		c.bundleFileUri = bundleFileUri[0]
	} else {
		c.bundleFileUri = DefaultBundleFileUri
	}

	return c
}

func (c *RuncMysql) SetDir(dir string) {
	c.dir = dir
}

func (c *RuncMysql) Load() error {
	err := os.MkdirAll(c.dir, os.ModePerm)
	if err != nil {
		return err
	}
	bundleFilePath := filepath.Join(c.dir, c.bundleZipFile)
	err = downLoadFile(c.bundleFileUri, bundleFilePath)
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
	os.Chdir(filepath.Join(c.dir, "bundle"))
	context := cli.NewContext(cli.NewApp(), MockFlagSet([]string{c.name}), nil)
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
	context := cli.NewContext(cli.NewApp(), MockFlagSet([]string{c.name}), nil)
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
	cmd := exec.Command("/bin/tar", "zxvf", bundleFilePath)
	return cmd.Run()
}
