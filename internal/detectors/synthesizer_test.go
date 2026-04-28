package detectors

import (
	"strings"
	"testing"
)

func TestMatch_IsPseudo(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"empty path is not pseudo", "", false},
		{"absolute unix path is real", "/Users/me/proj/.next", false},
		{"absolute root is real", "/", false},
		{"docker pseudo path", "docker:docker_images", true},
		{"colon-prefixed identifier", "kube:pods", true},
		{"relative dotted path is pseudo", ".terraform", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := (Match{Path: c.path}).IsPseudo(); got != c.want {
				t.Errorf("IsPseudo(%q) = %v, want %v", c.path, got, c.want)
			}
		})
	}
}

func TestDockerDetectors_ImplementSynthesizer(t *testing.T) {
	dockerCats := []Category{
		"docker_images",
		"docker_containers",
		"docker_build_cache",
		"docker_networks",
		"docker_volumes",
	}
	for _, id := range dockerCats {
		d := Default.Get(id)
		if d == nil {
			t.Errorf("%s not registered", id)
			continue
		}
		if _, ok := d.(Synthesizer); !ok {
			t.Errorf("%s does not implement Synthesizer", id)
		}
		if _, ok := d.(NativeOutputParser); !ok {
			t.Errorf("%s does not implement NativeOutputParser", id)
		}
	}
}

func TestParseDockerReclaimed(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int64
	}{
		{
			name:  "simple GB",
			input: "Total reclaimed space: 2.3GB\n",
			want:  2_300_000_000,
		},
		{
			name:  "MB lowercase",
			input: "deleted: x\nTotal reclaimed space: 512mb\n",
			want:  512_000_000,
		},
		{
			name:  "zero bytes",
			input: "Total reclaimed space: 0B\n",
			want:  0,
		},
		{
			name:  "kilobytes",
			input: "deleted: x\nTotal reclaimed space: 4.5KB\n",
			want:  4_500,
		},
		{
			name:  "TB upper bound",
			input: "Total reclaimed space: 1TB\n",
			want:  1_000_000_000_000,
		},
		{
			name:  "no parse line returns zero",
			input: "deleted x\ndeleted y\n",
			want:  0,
		},
		{
			name:  "empty input returns zero",
			input: "",
			want:  0,
		},
		{
			name:  "extra whitespace tolerated",
			input: "Total reclaimed space:    750MB",
			want:  750_000_000,
		},
		{
			name:  "docker builder prune format (Total:\\tX.YGB)",
			input: "ID\tRECLAIMABLE\nsha256:abc\t1.2GB\nTotal:\t2.5GB\n",
			want:  2_500_000_000,
		},
		{
			name:  "docker builder prune zero bytes",
			input: "Total:\t0B\n",
			want:  0,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseDockerReclaimed(c.input); got != c.want {
				t.Errorf("parseDockerReclaimed(%q) = %d, want %d", c.input, got, c.want)
			}
		})
	}
}

func TestParseDockerReclaimed_RejectsBogusFormat(t *testing.T) {
	bogus := []string{
		"Total reclaimed space: NaNGB",
		"reclaimed: 5GB",
		"Total reclaimed space: -1GB",
	}
	for _, in := range bogus {
		t.Run(strings.TrimSpace(in), func(t *testing.T) {
			if got := parseDockerReclaimed(in); got != 0 {
				t.Errorf("parseDockerReclaimed(%q) = %d, want 0", in, got)
			}
		})
	}
}
