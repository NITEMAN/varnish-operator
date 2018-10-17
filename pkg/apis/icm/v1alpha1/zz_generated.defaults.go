// +build !ignore_autogenerated

// Code generated by main. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&VarnishService{}, func(obj interface{}) { SetObjectDefaults_VarnishService(obj.(*VarnishService)) })
	scheme.AddTypeDefaultingFunc(&VarnishServiceList{}, func(obj interface{}) { SetObjectDefaults_VarnishServiceList(obj.(*VarnishServiceList)) })
	return nil
}

func SetObjectDefaults_VarnishService(in *VarnishService) {
	SetDefaults_VarnishService(in)
	SetDefaults_ServiceSpec(&in.Spec.Service)
	SetDefaults_VarnishDeployment(&in.Spec.Deployment)
}

func SetObjectDefaults_VarnishServiceList(in *VarnishServiceList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_VarnishService(a)
	}
}
