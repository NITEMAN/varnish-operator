package varnishadm

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

func TestNewVarnishAdministartor(t *testing.T) {
	cmd := NewVarnishAdministartor(
		90*time.Second,
		1*time.Second,
		"/etc/varnish",
		[]string{"-T", "127.0.0.1:6082", " ", "-S", "", "\t", "/etc/secret"},
	)
	expected := &VarnishAdm{
		binary:         "varnishadm",
		varnishAdmArgs: []string{"-T", "127.0.0.1:6082", "-S", "/etc/secret"},
		pingTimeout:    90 * time.Second,
		pingDelay:      1 * time.Second,
		vclBase:        "/etc/varnish",
		execute:        execCommandProvider,
	}
	if !cmp.Equal(cmd, expected, cmp.AllowUnexported(VarnishAdm{}), sameFunction) {
		t.Errorf("Unexpected response. Expected:\n%#v\n. Got:\n%#v\n Diff: %s\n", expected, cmd, cmp.Diff(expected, cmd, cmp.AllowUnexported(VarnishAdm{})))
	}
}

func TestPingCommand(t *testing.T) {
	cases := []struct {
		errExpected error
		timeout     time.Duration
		delay       time.Duration
		execute     executorProvider
		desc        string
	}{

		{
			nil,
			150 * time.Millisecond,
			5 * time.Microsecond,
			mockSuccessPing,
			"success",
		},
		{
			errors.New("varnish is unreachable"),
			150 * time.Millisecond,
			5 * time.Microsecond,
			mockUnreachabePing,
			"unreachable",
		},

		{
			nil,
			150 * time.Millisecond,
			5 * time.Microsecond,
			mockReachable5thTryPing,
			"reachable",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			p := &VarnishAdm{
				pingTimeout: tc.timeout,
				pingDelay:   tc.delay,
				execute:     tc.execute,
			}
			err := p.Ping()
			if !cmp.Equal(err, tc.errExpected, equalError) {
				tt.Errorf("Unexpected response for. %s", cmp.Diff(err, tc.errExpected))
			}
		})
	}

}

func TestReloadCommand(t *testing.T) {
	cases := []struct {
		errExpected error
		execute     executorProvider
		response    []byte
		desc        string
	}{
		{
			nil,
			mockSuccesResponse,
			[]byte("A response from external programm"),
			"success",
		},
		{
			errors.New("intermediate load err"),
			mockLoadErrResponse,
			[]byte("A response from external programm"),
			"errorOnLoad",
		},
		{
			errors.New("use error"),
			mockUseErrResponse,
			[]byte("A response from external programm"),
			"errorOnUse",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			p := &VarnishAdm{
				execute: tc.execute,
			}
			data, err := p.Reload("ver", "entry")
			if !cmp.Equal(data, tc.response) {
				tt.Errorf("Unexpected response %q\n Expected: %q", data, tc.response)
			}
			if !cmp.Equal(err, tc.errExpected, equalError) {
				tt.Errorf("Unexpected error return. %s", cmp.Diff(err, tc.errExpected))
			}
		})
	}
}

func TestListCommand(t *testing.T) {
	cases := []struct {
		errExpected error
		execute     executorProvider
		response    []byte
		desc        string
	}{
		{
			nil,
			mockSuccesResponse,
			[]byte("A response from external programm"),
			"success",
		},
		{
			errors.New("some error"),
			mockErrResponse,
			[]byte("A response from external programm"),
			"error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			p := &VarnishAdm{
				execute: tc.execute,
			}
			data, err := p.List()
			if !cmp.Equal(data, tc.response) {
				tt.Errorf("Unexpected response %q\n Expected: %q", data, tc.response)
			}
			if !cmp.Equal(err, tc.errExpected, equalError) {
				tt.Errorf("Unexpected error return. %s", cmp.Diff(err, tc.errExpected))
			}
		})
	}
}

func TestEnsureNotNilDefaultExecCommandProvider(t *testing.T) {
	c := execCommandProvider("echo", "hello", "world")
	if c == nil || (reflect.ValueOf(c).Kind() == reflect.Ptr && reflect.ValueOf(c).IsNil()) {
		t.Error("Unexpected nil for default execution command")
	}
}

// Mock the responder which count times reply with `intermediateErr` and after that
// return a `err`. In both cases it returns `response` as an output of external program.
// It take a pause durring `delay` for an intermediate responses.
type mockExecutor struct {
	response        []byte
	delay           time.Duration
	count           int
	err             error
	intermediateErr error
}

func (m *mockExecutor) CombinedOutput() ([]byte, error) {
	for m.count > 0 {
		time.Sleep(m.delay)
		m.count--
		return m.response, m.intermediateErr
	}
	return m.response, m.err
}

func mockSuccessPing(name string, args ...string) executor {
	return &mockExecutor{}
}

func mockUnreachabePing(name string, args ...string) executor {
	return &mockExecutor{err: errors.New("something goes wrong"), intermediateErr: errors.New("intermediate err")}
}

func mockReachable5thTryPing(name string, args ...string) executor {
	return &staticPingMock
}

func mockLoadErrResponse(name string, args ...string) executor {
	return &staticLoadErrMock
}

func mockUseErrResponse(name string, args ...string) executor {
	return &staticUseErrMock
}

func mockSuccesResponse(name string, args ...string) executor {
	return &mockExecutor{response: response}
}
func mockErrResponse(name string, args ...string) executor {
	return &mockExecutor{response: response, err: errors.New("some error")}
}

var (
	staticPingMock    = mockExecutor{count: 5, delay: 5 * time.Microsecond, intermediateErr: errors.New("intermediate err")}
	response          = []byte("A response from external programm")
	staticLoadErrMock = mockExecutor{count: 1, intermediateErr: errors.New("intermediate load err"), response: response}
	staticUseErrMock  = mockExecutor{count: 1, err: errors.New("use error"), response: response}
	simpleVCLconfig   = `
available   cold/cold          0 boot
active      auto/warm          0 v55329

`
	labeledVCLconfig = `
available   cold/cold          0 boot
active      auto/warm          0 v55329 (1 label)
available  label/warm          0 label1 -> v55329

`
	unknownVCLconfig = `
active   auto    warm         0    boot

`
	inactiveVCLconfig = `
available   cold/cold          0 boot
available     auto/cold          0 v55329

`
)

var (
	// equalError reports whether errors a and b are considered equal.
	// They're equal if both are nil, or both are not nil and a.Error() == b.Error().
	equalError = cmp.Comparer(func(a, b error) bool {
		return a == nil && b == nil || a != nil && b != nil && a.Error() == b.Error()
	})
	//there is noway to compare two function, but the trick allows to compare functions address
	sameFunction = cmp.Comparer(func(x, y executorProvider) bool {
		p1 := fmt.Sprintf("%v", x)
		p2 := fmt.Sprintf("%v", y)
		return p1 == p2
	})
)

func TestParseConfigs(t *testing.T) {
	cases := []struct {
		input       string
		expected    []VCLConfig
		expectedErr error
	}{
		{
			input: simpleVCLconfig,
			expected: []VCLConfig{
				{
					Status:      VCLStatusAvailable,
					Name:        "boot",
					Temperature: VCLTemperatureCold,
				},
				{
					Status:      VCLStatusActive,
					Name:        "v55329",
					Temperature: VCLTemperatureWarm,
				},
			},
			expectedErr: nil,
		},
		{
			input: labeledVCLconfig,
			expected: []VCLConfig{
				{
					Status:      VCLStatusAvailable,
					Name:        "boot",
					Temperature: VCLTemperatureCold,
				},
				{
					Status:      VCLStatusActive,
					Name:        "v55329",
					Temperature: VCLTemperatureWarm,
				},
				{
					Status:        VCLStatusAvailable,
					Name:          "label1",
					Temperature:   VCLTemperatureWarm,
					Label:         true,
					ReferencedVCL: proto.String("v55329"),
				},
			},
			expectedErr: nil,
		},
		{
			input:       unknownVCLconfig,
			expected:    nil,
			expectedErr: errors.New("unknown VarnishClusterVCL config format"),
		},
	}

	for _, c := range cases {
		actual, err := parseVCLConfigsList([]byte(c.input))
		if !cmp.Equal(err, c.expectedErr, equalError) {
			t.Logf("Unexpected error values: %#v. Expected: %#v", err, c.expectedErr)
			t.Fail()
		}
		if !cmp.Equal(actual, c.expected) {
			t.Logf(`
Input: 
%s
Parsed config:
%#v
Expected config:
%#v
`, c.input, actual, c.expected)
			t.Fail()
		}
	}
}

func TestGetActiveConfigurationName(t *testing.T) {
	cases := []struct {
		errExpected error
		execute     executorProvider
		response    string
		desc        string
	}{
		{
			nil,
			func(name string, args ...string) executor {
				return &mockExecutor{response: []byte(simpleVCLconfig)}
			},
			"v55329",
			"active",
		},
		{
			nil,
			func(name string, args ...string) executor {
				return &mockExecutor{response: []byte(labeledVCLconfig)}
			},
			"v55329",
			"active_labeled",
		},
		{
			errors.Errorf("No active VarnishClusterVCL configuration found"),
			func(name string, args ...string) executor {
				return &mockExecutor{response: []byte(inactiveVCLconfig)}
			},
			"",
			"inactive",
		},
		{
			errors.WithStack(errors.New("unknown VarnishClusterVCL config format")),
			func(name string, args ...string) executor {
				return &mockExecutor{response: []byte(unknownVCLconfig)}
			},
			"",
			"missformated",
		},
		{
			errors.Wrap(errors.New("some error"), string(response)),
			mockErrResponse,
			"",
			"externalError",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			p := &VarnishAdm{
				execute: tc.execute,
			}
			name, err := p.GetActiveConfigurationName()
			if !cmp.Equal(err, tc.errExpected, equalError) {
				tt.Logf("Unexpected error values: %#v. Expected: %#v", err, tc.errExpected)
				tt.Fail()
			}

			if !cmp.Equal(name, tc.response) {
				tt.Logf("Unexpected values: %#v. Expected: %#v", name, tc.response)
				tt.Fail()
			}
		})
	}
}
