package cli

import "testing"

func TestCheckMethod(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid method",
			args: args{
				s: "GetUser:GET::UserRequest{id:int32,name:string}:UserResponse{id:int32,name:string}",
			},
			wantErr: false,
		},
		{
			name: "invalid method",
			args: args{
				s: "GetUser:GET::UserRequest{id:int32,name:string}:UserResponse{id:int32,name:string",
			},
			wantErr: true,
		},
		{
			name: "invalid method name",
			args: args{
				s: "Get User:GET::UserRequest{id:int32,name:string}:UserResponse{id:int32,name:string}",
			},
			wantErr: true,
		},
		{
			name: "invalid method type",
			args: args{
				s: "GetUser:GETT::UserRequest{id:int32,name:string}:UserResponse{id:int32,name:string}",
			},
			wantErr: true,
		},
		{
			name: "invalid input or output",
			args: args{
				s: "GetUser:GET::UserRequest<id:int32,name:string}:UserResponse{id:int32,name:string",
			},
			wantErr: true,
		},
		{
			name: "valid input or output",
			args: args{
				s: "GetUser:GET::UserRequest{id:int32,name:string}:UserResponse{id:int32,name:string}",
			},
			wantErr: false,
		},
		{
			name: "valid multi input fields",
			args: args{
				s: "GetUser:GET::UserRequest{id:int32,name:string,age:int32,open:bool}:UserResponse{id:int32,name:string}",
			},
			wantErr: false,
		},
		{
			name: "invalid input",
			args: args{
				s: "GetUser:GET::UserRequest{id:int32,name:string",
			},
			wantErr: true,
		},
		{
			name: "invalid output",
			args: args{
				s: "GetUser:GET::UserRequest{id:int32,name:string}:UserResponse{id:int32,name:string",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckMethod(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("CheckMethod() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// FuzzCheckMethod is a fuzz test for CheckMethod
// func FuzzCheckMethod(f *testing.F) {
// 	const seed = "GetUser:GET:UserRequest{id:int32,name:string}:UserResponse{id:int32,name:string}"
// 	f.Add(len(seed), seed)

// 	f.Fuzz(func(t *testing.T, i int, s string) {
// 		if err := CheckMethod(s); err != nil {
// 			t.Errorf("CheckMethod(%s) error = %v", s, err)
// 		}
// 	})
// }
