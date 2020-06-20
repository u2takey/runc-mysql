package runc_mysql

import "testing"

func TestLoadFile(t *testing.T) {
	m := New("localhost:3306", "root")
	err := m.Load()
	if err != nil {
		t.Error(err)
	}

	err = m.Start()
	if err != nil {
		t.Error(err)
	}

}
