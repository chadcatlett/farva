package gateway

import (
	"strings"
	"testing"
)

type fakeReverseProxyConfigGetter struct {
	rc  reverseProxyConfig
	err error
}

func (fsm *fakeReverseProxyConfigGetter) ReverseProxyConfig() (*reverseProxyConfig, error) {
	return &fsm.rc, fsm.err
}

func TestRenderConfig(t *testing.T) {
	tests := []struct {
		rc   reverseProxyConfig
		want string
	}{
		// static code with and without message, no location
		{
			rc: reverseProxyConfig{
				HTTPServers: []httpReverseProxyServer{
					httpReverseProxyServer{
						ListenPort: 9001,
						StaticCode: 202,
					},
					httpReverseProxyServer{
						ListenPort:    9002,
						StaticCode:    203,
						StaticMessage: "ping pong",
					},
				},
			},
			want: `
pid /var/run/nginx.pid;
daemon on;

events {
    worker_connections 512;
}

http {
    server_names_hash_bucket_size 128;

    server {
        listen 9001;
        
        return 202;
    }

    server {
        listen 9002;
        
        return 203 'ping pong';
    }


}

stream {


}
`,
		},

		// single location, static code w/ and w/o message
		{
			rc: reverseProxyConfig{
				HTTPServers: []httpReverseProxyServer{
					httpReverseProxyServer{
						ListenPort: 9001,
						Locations: []httpReverseProxyLocation{
							httpReverseProxyLocation{
								Path:       "/foo",
								StaticCode: 202,
							},
						},
					},
					httpReverseProxyServer{
						ListenPort: 9002,
						Locations: []httpReverseProxyLocation{
							httpReverseProxyLocation{
								Path:          "/bar/baz",
								StaticCode:    203,
								StaticMessage: "ping pong",
							},
						},
					},
				},
			},
			want: `
pid /var/run/nginx.pid;
daemon on;

events {
    worker_connections 512;
}

http {
    server_names_hash_bucket_size 128;

    server {
        listen 9001;
        
        
        location /foo {
            return 202;
        }

    }

    server {
        listen 9002;
        
        
        location /bar/baz {
            return 203 'ping pong';
        }

    }


}

stream {


}
`,
		},

		// one HTTP server, multiple upstreams and paths
		{
			rc: reverseProxyConfig{
				HTTPServers: []httpReverseProxyServer{
					httpReverseProxyServer{
						ListenPort: 9001,
						Locations: []httpReverseProxyLocation{
							httpReverseProxyLocation{
								Path:     "/abc",
								Upstream: "foo",
							},
							httpReverseProxyLocation{
								Path:     "/def",
								Upstream: "bar",
							},
						},
					},
				},
				HTTPUpstreams: []httpReverseProxyUpstream{
					httpReverseProxyUpstream{
						Name: "foo",
						Servers: []reverseProxyUpstreamServer{
							reverseProxyUpstreamServer{
								Name: "ping",
								Host: "ping.example.com",
								Port: 443,
							},
							reverseProxyUpstreamServer{
								Name: "pong",
								Host: "pong.example.com",
								Port: 80,
							},
						},
					},
					httpReverseProxyUpstream{
						Name: "bar",
						Servers: []reverseProxyUpstreamServer{
							reverseProxyUpstreamServer{
								Name: "ding",
								Host: "ding.example.com",
								Port: 443,
							},
							reverseProxyUpstreamServer{
								Name: "dong",
								Host: "dong.example.com",
								Port: 80,
							},
						},
					},
				},
			},
			want: `
pid /var/run/nginx.pid;
daemon on;

events {
    worker_connections 512;
}

http {
    server_names_hash_bucket_size 128;

    server {
        listen 9001;
        
        
        location /abc {
            
            proxy_pass http://foo;
        }

        location /def {
            
            proxy_pass http://bar;
        }

    }


    upstream foo {

        server ping.example.com:443;  # ping
        server pong.example.com:80;  # pong
    }

    upstream bar {

        server ding.example.com:443;  # ding
        server dong.example.com:80;  # dong
    }

}

stream {


}
`,
		},

		// two TCP servers, various upstreams
		{
			rc: reverseProxyConfig{
				TCPServers: []tcpReverseProxyServer{
					tcpReverseProxyServer{
						ListenPort: 9001,
						Upstream:   "foo",
					},
					tcpReverseProxyServer{
						ListenPort: 9002,
						Upstream:   "bar",
					},
				},
				TCPUpstreams: []tcpReverseProxyUpstream{
					tcpReverseProxyUpstream{
						Name: "foo",
						Servers: []reverseProxyUpstreamServer{
							reverseProxyUpstreamServer{
								Name: "ping",
								Host: "ping.example.com",
								Port: 443,
							},
						},
					},
					tcpReverseProxyUpstream{
						Name: "bar",
						Servers: []reverseProxyUpstreamServer{
							reverseProxyUpstreamServer{
								Name: "ding",
								Host: "ding.example.com",
								Port: 443,
							},
							reverseProxyUpstreamServer{
								Name: "dong",
								Host: "dong.example.com",
								Port: 80,
							},
						},
					},
				},
			},
			want: `
pid /var/run/nginx.pid;
daemon on;

events {
    worker_connections 512;
}

http {
    server_names_hash_bucket_size 128;


}

stream {

    server {
        listen 9001;
        proxy_pass foo;
    }

    server {
        listen 9002;
        proxy_pass bar;
    }


    upstream foo {

        server ping.example.com:443;  # ping
    }

    upstream bar {

        server ding.example.com:443;  # ding
        server dong.example.com:80;  # dong
    }

}
`,
		},
	}

	for i, tt := range tests {
		fsm := fakeReverseProxyConfigGetter{rc: tt.rc}
		cfg := DefaultNGINXConfig
		got, err := renderConfig(&cfg, &fsm.rc)
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
			continue
		}

		if tt.want != string(got) {
			wantPretty := strings.Replace(tt.want, " ", "÷", -1)
			gotPretty := strings.Replace(string(got), " ", "÷", -1)
			t.Errorf("case %d: unexpected output: want=%sgot=%s", i, wantPretty, gotPretty)
		}
	}
}