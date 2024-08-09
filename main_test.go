package main

import (
	"reflect"
	"testing"
)

func TestAddUserNamePosScore(t *testing.T) {
	tests := []struct {
		name  string
		input []ToDo
		want  []ToDo
	}{
		{
			name: "user order",
			input: []ToDo{
				{
					Body: "@hiromu @tarou @jiro @saburo @shiro @gorou @rokuro @nanaro @hatiro @kuro",
				},
				{
					Body: "@tarou @jiro @saburo @hiromu @shiro @gorou @rokuro @nanaro @hatiro @kuro",
				},
			},
			want: []ToDo{
				{
					Body:  "@hiromu @tarou @jiro @saburo @shiro @gorou @rokuro @nanaro @hatiro @kuro",
					Score: 35,
				},
				{
					Body:  "@tarou @jiro @saburo @hiromu @shiro @gorou @rokuro @nanaro @hatiro @kuro",
					Score: 26,
				},
			},
		},
		{
			name: "users num",
			input: []ToDo{
				{
					Body: "@tarou @jiro @saburo @hiromu @shiro @gorou @rokuro @nanaro @hatiro @kuro",
				},
				{
					Body: "@tarou @jiro @saburo @hiromu @shiro @gorou @rokuro @nanaro",
				},
				{
					Body: "@saburo @hiromu @shiro @gorou",
				},
				{
					Body: "@hiromu",
				},
			},
			want: []ToDo{
				{
					Body:  "@tarou @jiro @saburo @hiromu @shiro @gorou @rokuro @nanaro @hatiro @kuro",
					Score: 26,
				},
				{
					Body:  "@tarou @jiro @saburo @hiromu @shiro @gorou @rokuro @nanaro",
					Score: 25,
				},
				{
					Body:  "@saburo @hiromu @shiro @gorou",
					Score: 35,
				},
				{
					Body:  "@hiromu",
					Score: 80,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addUserNamePosScore(tt.input, "hiromu")
			if !reflect.DeepEqual(tt.input, tt.want) {
				t.Errorf("\ngot : %v\nwant: %v", tt.input, tt.want)
			}
		})
	}
}
