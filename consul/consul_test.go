package consul

import (
	"testing"
)

func TestNewHandler(t *testing.T) {
	client, err := NewHandler()
	if err != nil || client != nil {
		t.Fail()
	}
}

func TestPut(t *testing.T) {
	client, _ := NewHandler()
	ok, err := client.Put("/var/git2consul/test/data", []byte("testdata"))
	if !ok {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
}

func TestRead(t *testing.T) {
	client, _ := NewHandler()
	data, err := client.read("/var/git2consul/test/data")
	if string(data) != string([]byte("testdata")) {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
}

func TestIsExist(t *testing.T) {
	client, _ := NewHandler()
	_, err := client.Put("/var/git2consul/test/data", []byte("testdata"))
	if err != nil {
		t.Error(err)
	}
	ok, err := client.IsExist("/var/git2consul/test/data")
	if !ok {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}

}
