package httpapi

import "net/http"

type MethodGet struct{}

func (*MethodGet) GetMethod() string { return http.MethodGet }

type MethodPost struct{}

func (*MethodPost) GetMethod() string { return http.MethodPost }

type MethodPut struct{}

func (*MethodPut) GetMethod() string { return http.MethodPut }

type MethodDelete struct{}

func (*MethodDelete) GetMethod() string { return http.MethodDelete }

type MethodPatch struct{}

func (*MethodPatch) GetMethod() string { return http.MethodPatch }
