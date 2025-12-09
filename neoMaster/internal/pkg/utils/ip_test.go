package utils

import (
	"net"
	"reflect"
	"testing"
)

func TestCIDR2IPs(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		want    []string
		wantErr bool
	}{
		{
			name:    "valid_cidr_30",
			cidr:    "192.168.0.0/30",
			want:    []string{"192.168.0.0", "192.168.0.1", "192.168.0.2", "192.168.0.3"},
			wantErr: false,
		},
		{
			name:    "invalid_cidr",
			cidr:    "192.168.0.1",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid_ipv6",
			cidr:    "2001:db8::/32",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CIDR2IPs(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("CIDR2IPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CIDR2IPs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRange2IPs(t *testing.T) {
	tests := []struct {
		name     string
		rangeStr string
		want     []string
		wantErr  bool
	}{
		{
			name:     "valid_range",
			rangeStr: "192.168.0.1-192.168.0.3",
			want:     []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"},
			wantErr:  false,
		},
		{
			name:     "invalid_format",
			rangeStr: "192.168.0.1",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "invalid_start_ip",
			rangeStr: "300.0.0.1-192.168.0.3",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "start_greater_than_end",
			rangeStr: "192.168.0.5-192.168.0.1",
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Range2IPs(tt.rangeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Range2IPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Range2IPs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIP2IntAndInt2IP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want uint32
	}{
		{
			name: "0.0.0.0",
			ip:   "0.0.0.0",
			want: 0,
		},
		{
			name: "192.168.1.1",
			ip:   "192.168.1.1",
			want: 3232235777,
		},
		{
			name: "255.255.255.255",
			ip:   "255.255.255.255",
			want: 4294967295,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			gotInt := IP2Int(ip)
			if gotInt != tt.want {
				t.Errorf("IP2Int() = %v, want %v", gotInt, tt.want)
			}
			gotIP := Int2IP(gotInt)
			if !gotIP.Equal(ip) {
				t.Errorf("Int2IP() = %v, want %v", gotIP, ip)
			}
		})
	}
}

func TestMergeIPs(t *testing.T) {
	tests := []struct {
		name string
		ips  []string
		want []string
	}{
		{
			name: "normal_merge",
			ips:  []string{"192.168.1.2", "192.168.1.1", "192.168.1.2"},
			want: []string{"192.168.1.1", "192.168.1.2"},
		},
		{
			name: "empty_list",
			ips:  []string{},
			want: []string{},
		},
		{
			name: "with_empty_strings",
			ips:  []string{"192.168.1.1", "", "  ", "192.168.1.2"},
			want: []string{"192.168.1.1", "192.168.1.2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeIPs(tt.ips)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeIPs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseIPPairs(t *testing.T) {
	tests := []struct {
		name     string
		rangeStr string
		want     []string
		wantErr  bool
	}{
		{
			name:     "full_range",
			rangeStr: "192.168.0.1-192.168.0.3",
			want:     []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"},
			wantErr:  false,
		},
		{
			name:     "short_range_last_byte",
			rangeStr: "192.168.0.1-3",
			want:     []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"},
			wantErr:  false,
		},
		{
			name:     "short_range_last_two_bytes",
			rangeStr: "192.168.0.254-1.1",
			want:     []string{"192.168.0.254", "192.168.0.255", "192.168.1.0", "192.168.1.1"},
			wantErr:  false,
		},
		{
			name:     "invalid_range_order",
			rangeStr: "192.168.0.5-1",
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIPPairs(tt.rangeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIPPairs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseIPPairs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidators(t *testing.T) {
	t.Run("IsURL", func(t *testing.T) {
		validURLs := []string{
			"http://example.com",
			"https://example.com/path",
			"ftp://example.com:21",
			"http://192.168.1.1",
			"http://[::1]",
		}
		for _, u := range validURLs {
			if !IsURL(u) {
				t.Errorf("IsURL(%q) = false, want true", u)
			}
		}

		invalidURLs := []string{
			"not_a_url",
			"example.com", // No scheme
			"://example.com",
		}
		for _, u := range invalidURLs {
			if IsURL(u) {
				t.Errorf("IsURL(%q) = true, want false", u)
			}
		}
	})

	t.Run("IsIPPort", func(t *testing.T) {
		valid := []string{
			"127.0.0.1:8080",
			"[::1]:80",
			"192.168.1.1:65535",
		}
		for _, s := range valid {
			if !IsIPPort(s) {
				t.Errorf("IsIPPort(%q) = false, want true", s)
			}
		}

		invalid := []string{
			"example.com:8080",
			"127.0.0.1",
			"127.0.0.1:70000", // Port too large
			"256.0.0.1:80",    // Invalid IP
		}
		for _, s := range invalid {
			if IsIPPort(s) {
				t.Errorf("IsIPPort(%q) = true, want false", s)
			}
		}
	})

	t.Run("IsDomainPort", func(t *testing.T) {
		valid := []string{
			"example.com:80",
			"sub.example.co.uk:443",
			"localhost:8080",
		}
		for _, s := range valid {
			if !IsDomainPort(s) {
				t.Errorf("IsDomainPort(%q) = false, want true", s)
			}
		}

		invalid := []string{
			"127.0.0.1:80",
			"example.com",
			"example.com:70000",
		}
		for _, s := range invalid {
			if IsDomainPort(s) {
				t.Errorf("IsDomainPort(%q) = true, want false", s)
			}
		}
	})

	t.Run("IsNetlocPort", func(t *testing.T) {
		valid := []string{
			"127.0.0.1:80",
			"example.com:80",
			"[::1]:80",
		}
		for _, s := range valid {
			if !IsNetlocPort(s) {
				t.Errorf("IsNetlocPort(%q) = false, want true", s)
			}
		}
	})

	t.Run("IsCIDR", func(t *testing.T) {
		valid := []string{
			"192.168.1.0/24",
			"10.0.0.0/8",
			"::1/128",
		}
		for _, s := range valid {
			if !IsCIDR(s) {
				t.Errorf("IsCIDR(%q) = false, want true", s)
			}
		}

		invalid := []string{
			"192.168.1.1",
			"192.168.1.0/33",
			"invalid",
		}
		for _, s := range invalid {
			if IsCIDR(s) {
				t.Errorf("IsCIDR(%q) = true, want false", s)
			}
		}
	})

	t.Run("IsIP_IsIPv4_IsIPv6", func(t *testing.T) {
		v4 := "192.168.1.1"
		v6 := "2001:db8::1"
		invalid := "256.0.0.1"

		if !IsIP(v4) || !IsIPv4(v4) || IsIPv6(v4) {
			t.Errorf("IPv4 check failed for %s", v4)
		}
		if !IsIP(v6) || IsIPv4(v6) || !IsIPv6(v6) {
			t.Errorf("IPv6 check failed for %s", v6)
		}
		if IsIP(invalid) {
			t.Errorf("Invalid IP check failed for %s", invalid)
		}
	})

	t.Run("IsPort", func(t *testing.T) {
		valid := []string{"0", "80", "65535"}
		invalid := []string{"-1", "65536", "abc", ""}

		for _, p := range valid {
			if !IsPort(p) {
				t.Errorf("IsPort(%q) = false, want true", p)
			}
		}
		for _, p := range invalid {
			if IsPort(p) {
				t.Errorf("IsPort(%q) = true, want false", p)
			}
		}
	})

	t.Run("IsDomain", func(t *testing.T) {
		valid := []string{
			"example.com",
			"sub.domain.co.uk",
			"localhost",
			"a.b.c.d.com",
		}
		for _, d := range valid {
			if !IsDomain(d) {
				t.Errorf("IsDomain(%q) = false, want true", d)
			}
		}

		invalid := []string{
			"127.0.0.1",
			"-example.com",
			"example.com-",
			"",
		}
		for _, d := range invalid {
			if IsDomain(d) {
				t.Errorf("IsDomain(%q) = true, want false", d)
			}
		}
	})
}
