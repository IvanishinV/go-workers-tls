package workers

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
)

type config struct {
	processId    string
	Namespace    string
	PollInterval int
	Pool         *redis.Pool
	Fetch        func(queue string) Fetcher
}

var Config *config

func Configure(options map[string]string) {
	var poolSize int
	var namespace string
	var pollInterval int

	if options["server"] == "" {
		panic("Configure requires a 'server' option, which identifies a Redis instance")
	}
	if options["process"] == "" {
		panic("Configure requires a 'process' option, which uniquely identifies this instance")
	}
	if options["pool"] == "" {
		options["pool"] = "1"
	}
	if options["namespace"] != "" {
		namespace = options["namespace"] + ":"
	}
	if seconds, err := strconv.Atoi(options["poll_interval"]); err == nil {
		pollInterval = seconds
	} else {
		pollInterval = 15
	}

	poolSize, _ = strconv.Atoi(options["pool"])

	var tlsConfig *tls.Config
	if options["tls"] == "true" {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}

		if options["tls_skip_verify"] == "true" {
			tlsConfig.InsecureSkipVerify = true
		}

		if options["tls_cert"] != "" && options["tls_key"] != "" {
			cert, err := tls.LoadX509KeyPair(options["tls_cert"], options["tls_key"])
			if err != nil {
				panic(fmt.Sprintf("failed to load client certificate: %v", err))
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		if options["tls_ca"] != "" {
			caCert, err := os.ReadFile(options["tls_ca"])
			if err != nil {
				panic(fmt.Sprintf("failed to read CA file: %v", err))
			}
			caPool := x509.NewCertPool()
			if ok := caPool.AppendCertsFromPEM(caCert); !ok {
				panic("failed to append CA certs: no valid PEM data found")
			}
			tlsConfig.RootCAs = caPool
		}

		if options["tls_skip_verify"] != "true" {
			if host, _, err := net.SplitHostPort(options["server"]); err == nil {
				tlsConfig.ServerName = host
			}
		}
	}

	Config = &config{
		options["process"],
		namespace,
		pollInterval,
		&redis.Pool{
			MaxIdle:     poolSize,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				var c redis.Conn
				var err error

				if tlsConfig != nil {
					c, err = redis.Dial(
						"tcp",
						options["server"],
						redis.DialUseTLS(true),
						redis.DialTLSConfig(tlsConfig),
					)
				} else {
					c, err = redis.Dial("tcp", options["server"])
				}
				if err != nil {
					return nil, err
				}

				if options["password"] != "" {
					if _, err := c.Do("AUTH", options["password"]); err != nil {
						c.Close()
						return nil, err
					}
				}
				if options["database"] != "" {
					if _, err := c.Do("SELECT", options["database"]); err != nil {
						c.Close()
						return nil, err
					}
				}
				return c, nil
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
		},
		func(queue string) Fetcher {
			return NewFetch(queue, make(chan *Msg), make(chan bool))
		},
	}
}
