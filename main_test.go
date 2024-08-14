package main

import (
	"reflect"
	"testing"
)

func TestAddUserNamePosScore(t *testing.T) {
	tests := []struct {
		name  string
		input []ToDo
		want  map[int]float64
	}{
		{
			name: "user order",
			input: []ToDo{
				{
					Body: "@hiromu @tarou @jiro @saburo @shiro @gorou @rokuro @nanaro @hatiro @kuro",
					Target: Target{
						IID: 1,
					},
				},
				{
					Body: "@tarou @jiro @saburo @hiromu @shiro @gorou @rokuro @nanaro @hatiro @kuro",
					Target: Target{
						IID: 2,
					},
				},
			},
			want: map[int]float64{
				1: 35,
				2: 26,
			},
		},
		{
			name: "users num",
			input: []ToDo{
				{
					Body: "@tarou @jiro @saburo @hiromu @shiro @gorou @rokuro @nanaro @hatiro @kuro",
					Target: Target{
						IID: 1,
					},
				},
				{
					Body: "@tarou @jiro @saburo @hiromu @shiro @gorou @rokuro @nanaro",
					Target: Target{
						IID: 2,
					},
				},
				{
					Body: "@saburo @hiromu @shiro @gorou",
					Target: Target{
						IID: 3,
					},
				},
				{
					Body: "@hiromu",
					Target: Target{
						IID: 4,
					},
				},
			},
			want: map[int]float64{
				1: 26,
				2: 25,
				3: 35,
				4: 80,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := userNamePosScore(tt.input, "hiromu")
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("\ngot : %v\nwant: %v", tt.input, tt.want)
			}
		})
	}
}
