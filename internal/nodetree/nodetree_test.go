package nodetree

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/getsentry/vroom/internal/frame"
	"github.com/getsentry/vroom/internal/testutil"
)

func TestNodeTreeCollapse(t *testing.T) {
	tests := []struct {
		name string
		node *Node
		want []*Node
	}{
		{
			name: "single node no collapse",
			node: &Node{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children:      []*Node{},
			},
			want: []*Node{{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				StartNS:       0,
				SampleCount:   10,
				Children:      []*Node{},
			}},
		},
		{
			name: "multiple children no collapse",
			node: &Node{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    5,
						EndNS:         5,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child1",
						Package:       "package",
						Path:          "path",
						SampleCount:   5,
						StartNS:       0,
						Children:      []*Node{},
					},
					{
						DurationNS:    5,
						EndNS:         10,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child2",
						Package:       "package",
						Path:          "path",
						SampleCount:   5,
						StartNS:       5,
						Children:      []*Node{},
					},
				},
			},
			want: []*Node{{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    5,
						EndNS:         5,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child1",
						Package:       "package",
						Path:          "path",
						SampleCount:   5,
						StartNS:       0,
						Children:      []*Node{},
					},
					{
						DurationNS:    5,
						EndNS:         10,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child2",
						Package:       "package",
						Path:          "path",
						SampleCount:   5,
						StartNS:       5,
						Children:      []*Node{},
					},
				},
			}},
		},
		{
			name: "collapse single sample",
			node: &Node{
				DurationNS:    4,
				EndNS:         4,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   4,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    1,
						EndNS:         1,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child1",
						Package:       "package",
						Path:          "path",
						SampleCount:   1,
						StartNS:       0,
						Children:      []*Node{},
					},
					{
						DurationNS:    1,
						EndNS:         2,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child2",
						Package:       "package",
						Path:          "path",
						SampleCount:   1,
						StartNS:       1,
						Children:      []*Node{},
					},
					{
						DurationNS:    2,
						EndNS:         4,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child3",
						Package:       "package",
						Path:          "path",
						SampleCount:   2,
						StartNS:       2,
						Children:      []*Node{},
					},
				},
			},
			want: []*Node{{
				DurationNS:    4,
				EndNS:         4,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   4,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    2,
						EndNS:         4,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child3",
						Package:       "package",
						Path:          "path",
						SampleCount:   2,
						StartNS:       2,
						Children:      []*Node{},
					},
				},
			}},
		},
		{
			name: "single child no collapse - duration mismatch",
			node: &Node{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    5,
						EndNS:         5,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child",
						Package:       "package",
						Path:          "path",
						SampleCount:   5,
						StartNS:       0,
						Children:      []*Node{},
					},
				},
			},
			want: []*Node{{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    5,
						EndNS:         5,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child",
						Package:       "package",
						Path:          "path",
						SampleCount:   5,
						StartNS:       0,
						Children:      []*Node{},
					},
				},
			}},
		},
		{
			name: "single child collapse parent because child is application",
			node: &Node{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    10,
						EndNS:         10,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child",
						Package:       "package",
						Path:          "path",
						SampleCount:   10,
						StartNS:       0,
						Children:      []*Node{},
					},
				},
			},
			want: []*Node{{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "child",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children:      []*Node{},
			}},
		},
		{
			name: "single child collapse parent because both system application",
			node: &Node{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: false,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    10,
						EndNS:         10,
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child",
						Package:       "package",
						Path:          "path",
						SampleCount:   10,
						StartNS:       0,
						Children:      []*Node{},
					},
				},
			},
			want: []*Node{{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: false,
				Line:          0,
				Name:          "child",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children:      []*Node{},
			}},
		},
		{
			name: "single child collapse child because parent is application",
			node: &Node{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    10,
						EndNS:         10,
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child",
						Package:       "package",
						Path:          "path",
						SampleCount:   10,
						StartNS:       0,
						Children:      []*Node{},
					},
				},
			},
			want: []*Node{{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children:      []*Node{},
			}},
		},
		{
			name: "nested nodes, all unknown name",
			node: &Node{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "",
				Package:       "",
				Path:          "",
				SampleCount:   1,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    5,
						EndNS:         5,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "",
						Package:       "",
						Path:          "",
						SampleCount:   1,
						StartNS:       0,
						Children: []*Node{
							{
								DurationNS:    5,
								EndNS:         5,
								Fingerprint:   0,
								IsApplication: true,
								Line:          0,
								Name:          "",
								Package:       "",
								Path:          "",
								SampleCount:   1,
								StartNS:       0,
								Children: []*Node{
									{
										DurationNS:    5,
										EndNS:         5,
										Fingerprint:   0,
										IsApplication: false,
										Line:          0,
										Name:          "",
										Package:       "",
										Path:          "",
										SampleCount:   1,
										StartNS:       0,
										Children:      []*Node{},
									},
								},
							},
						},
					},
					{
						DurationNS:    10,
						EndNS:         10,
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "",
						Package:       "",
						Path:          "",
						SampleCount:   1,
						StartNS:       5,
						Children:      []*Node{},
					},
				},
			},
			want: []*Node{},
		},
		{
			name: "collapse deeply nested node",
			node: &Node{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    5,
						EndNS:         5,
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child1-1",
						Package:       "package",
						Path:          "path",
						SampleCount:   5,
						StartNS:       0,
						Children: []*Node{
							{
								DurationNS:    5,
								EndNS:         5,
								Fingerprint:   0,
								IsApplication: true,
								Line:          0,
								Name:          "child2-1",
								Package:       "package",
								Path:          "path",
								SampleCount:   5,
								StartNS:       0,
								Children: []*Node{
									{
										DurationNS:    5,
										EndNS:         5,
										Fingerprint:   0,
										IsApplication: false,
										Line:          0,
										Name:          "child3-1",
										Package:       "package",
										Path:          "path",
										SampleCount:   5,
										StartNS:       0,
										Children:      []*Node{},
									},
								},
							},
						},
					},
					{
						DurationNS:    5,
						EndNS:         10,
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child1-2",
						Package:       "package",
						Path:          "path",
						SampleCount:   5,
						StartNS:       5,
						Children: []*Node{
							{
								DurationNS:    5,
								EndNS:         10,
								Fingerprint:   0,
								IsApplication: true,
								Line:          0,
								Name:          "",
								Package:       "",
								Path:          "",
								SampleCount:   5,
								StartNS:       5,
								Children: []*Node{
									{
										DurationNS:    5,
										EndNS:         10,
										Fingerprint:   0,
										IsApplication: false,
										Line:          0,
										Name:          "child3-1",
										Package:       "package",
										Path:          "path",
										SampleCount:   5,
										StartNS:       5,
										Children:      []*Node{},
									},
								},
							},
						},
					},
				},
			},
			want: []*Node{{
				DurationNS:    10,
				EndNS:         10,
				Fingerprint:   0,
				IsApplication: true,
				Line:          0,
				Name:          "root",
				Package:       "package",
				Path:          "path",
				SampleCount:   10,
				StartNS:       0,
				Children: []*Node{
					{
						DurationNS:    5,
						EndNS:         5,
						Fingerprint:   0,
						IsApplication: true,
						Line:          0,
						Name:          "child2-1",
						Package:       "package",
						Path:          "path",
						SampleCount:   5,
						StartNS:       0,
						Children:      []*Node{},
					},
					{
						DurationNS:    5,
						EndNS:         10,
						Fingerprint:   0,
						IsApplication: false,
						Line:          0,
						Name:          "child3-1",
						Package:       "package",
						Path:          "path",
						SampleCount:   5,
						StartNS:       5,
						Children:      []*Node{},
					},
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.node.Collapse()
			if diff := testutil.Diff(result, tt.want); diff != "" {
				t.Fatalf("Result mismatch: got - want +\n%s", diff)
			}
		})
	}
}

func TestFramePackageName(t *testing.T) {
	var f frame.Frame
	err := json.Unmarshal([]byte(`{
		"function": "_dispatch_event_loop_leave_immediate",
		"in_app": false,
		"instruction_addr": "0x1c44cfe34",
		"package": "/usr/lib/system/libdispatch.dylib",
		"status": "symbolicated",
		"sym_addr": "0x1c44cfd60",
		"symbol": "_dispatch_event_loop_leave_immediate"
	}`), &f)
	if err != nil {
		t.Fatal(err)
	}
	n := NodeFromFrame(f, 0, 0, 1234567890)
	if strings.HasPrefix(n.Package, "/") || n.Package != f.PackageBaseName() {
		t.Fatal("package name should not be a path")
	}
}
