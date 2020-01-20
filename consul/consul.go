package consul

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

//ConsulHandler interacts with the consul client
type ConsulHandler struct {
	Client *api.Client
	opts   consuloptions
}

//Lock generates a session for cron
func (c *ConsulHandler) Lock(key string) <-chan struct{} {
	stopCh := make(chan struct{})
	lock, err := c.Client.LockKey(key)
	if err != nil {
		return nil
	}
	lockCh, err := lock.Lock(stopCh)
	if err != nil {
		panic(err)
	}
	return lockCh
}

//Unlock runs until successful
func (c *ConsulHandler) Unlock(key string) bool {
	lock, err := c.Client.LockKey(key)
	if err != nil {
		logrus.Warningln(err)
		return false
	}
	for {
		if err := lock.Unlock(); err != nil {
			logrus.Warningf("Error occured unlocking %v", err)
			continue
		}
		break
	}
	return true
}

var defaultConsulOptions = consuloptions{
	Timeout:               30 * time.Second,
	KeepAlive:             30 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 10 * time.Second,
	ExpectContinueTimeout: 2 * time.Second,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   100,
	DisableCompression:    true,
	InsecureSkipVerify:    true,
	Scheme:                "http",
}

//NewHandler for interacting with consul client
func NewHandler(opt ...ConsulOption) (consulHandler *ConsulHandler, err error) {
	opts := defaultConsulOptions
	for _, f := range opt {
		err := f(&opts)
		if err != nil {
			return nil, errors.Wrap(err, "error setting option")
		}
	}
	client, err := api.NewClient(opts.Config)
	if err != nil {
		return nil, err
	}
	consulHandler = &ConsulHandler{
		Client: client,
		opts:   opts,
	}
	return
}

type consuloptions struct {
	Timeout               time.Duration
	KeepAlive             time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	ExpectContinueTimeout time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	DisableCompression    bool
	InsecureSkipVerify    bool
	Config                *api.Config
	Scheme                string
	WriteOptions          *api.WriteOptions
	QueryOptions          *api.QueryOptions
}

//ConsulOption decorator
type ConsulOption func(*consuloptions) error

//Config sets consul options for client
func Config(address, token string) ConsulOption {
	return func(o *consuloptions) error {
		if token != "" {
			o.Config = &api.Config{
				Address:   address,
				Scheme:    o.Scheme,
				Transport: TransportConfig(o),
				Token:     token,
			}
		}
		o.Config = &api.Config{
			Address:   address,
			Scheme:    o.Scheme,
			Transport: TransportConfig(o),
		}
		return nil
	}
}

//Timeout sets value for http transport
func Timeout(t time.Duration) ConsulOption {
	return func(o *consuloptions) error {
		o.Timeout = t
		return nil
	}
}

//KeepAlive sets keep alive for http transport
func KeepAlive(keepAlive time.Duration) ConsulOption {
	return func(o *consuloptions) error {
		o.KeepAlive = keepAlive
		return nil
	}
}

//TLSHandShakeTimeOut sets tls timeout for conig
func TLSHandShakeTimeOut(tlsTimeOut time.Duration) ConsulOption {
	return func(o *consuloptions) error {
		o.TLSHandshakeTimeout = tlsTimeOut
		return nil
	}
}

//ResponseHeaderTimeOut for http transport
func ResponseHeaderTimeOut(respHeaderTimeOut time.Duration) ConsulOption {
	return func(o *consuloptions) error {
		o.ResponseHeaderTimeout = respHeaderTimeOut
		return nil
	}
}

//ExpectContinueTimeout sets a time out for exepected continue
func ExpectContinueTimeout(expectContinueTimeout time.Duration) ConsulOption {
	return func(o *consuloptions) error {
		o.ExpectContinueTimeout = expectContinueTimeout
		return nil
	}
}

//MaxIdleConns sets max ideal connections for host as well
func MaxIdleConns(maxIdleConns int) ConsulOption {
	return func(o *consuloptions) error {
		o.MaxIdleConns = maxIdleConns
		o.MaxIdleConnsPerHost = maxIdleConns
		return nil
	}
}

//DisableCompression sets option http transport config
func DisableCompression(disableCompression bool) ConsulOption {
	return func(o *consuloptions) error {
		o.DisableCompression = disableCompression
		return nil
	}
}

//InsecureSkipVerify sets option for TLS Config
func InsecureSkipVerify(insecureSkipVerify bool) ConsulOption {
	return func(o *consuloptions) error {
		o.InsecureSkipVerify = insecureSkipVerify
		return nil
	}
}

//TransportConfig sets options for http transport
func TransportConfig(o *consuloptions) *http.Transport {
	return &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   o.Timeout,
			KeepAlive: o.KeepAlive,
		}).Dial,
		TLSHandshakeTimeout:   o.TLSHandshakeTimeout,
		ResponseHeaderTimeout: o.ResponseHeaderTimeout,
		ExpectContinueTimeout: o.ExpectContinueTimeout,
		MaxIdleConns:          o.MaxIdleConns,
		MaxIdleConnsPerHost:   o.MaxIdleConnsPerHost,
		DisableCompression:    o.DisableCompression,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: o.InsecureSkipVerify,
		},
	}
}

//IsExist tells you where content exist or not at that path
func (c *ConsulHandler) IsExist(path string) (bool, error) {
	kv := c.Client.KV()
	_, _, err := kv.Get(path, &api.QueryOptions{
		Token: c.opts.Config.Token,
	})
	if err != nil {
		return false, err
	}
	return true, err
}

func (c *ConsulHandler) read(path string) ([]byte, error) {
	kv := c.Client.KV()
	kvPair, _, err := kv.Get(path, c.opts.QueryOptions)
	if err != nil {
		return nil, err
	}
	return kvPair.Value, nil
}

//Put content in Consul
func (c *ConsulHandler) Put(path string, value []byte) (bool, error) {
	consulKV := c.Client.KV()
	p := &api.KVPair{Key: path, Value: value}
	_, err := consulKV.Put(p, &api.WriteOptions{Token: c.opts.Config.Token})
	if err != nil {
		return false, err
	}
	return true, nil
}

//Delete a key from consul
func (c *ConsulHandler) Delete(path string) (bool, error) {
	_, err := c.Client.KV().Delete(path, &api.WriteOptions{Token: c.opts.Config.Token})
	if err != nil {
		return false, err
	}
	return true, nil
}

//ServiceRegistration registers a service by name
func (c *ConsulHandler) ServiceRegistration(name string, tags ...string) error {
	return c.Client.Agent().ServiceRegister(&api.AgentServiceRegistration{Name: name, Tags: tags})
}

//ServiceDeregistation deregisters a service by name
func (c *ConsulHandler) ServiceDeregistation(name string) error {
	return c.Client.Agent().ServiceDeregister(name)
}
