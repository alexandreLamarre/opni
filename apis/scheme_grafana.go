//go:build !minimal

package apis

import (
	//opnigrafanav1alpha1 "github.com/rancher/opni/apis/grafana/v1alpha1"
	opnigrafanav1beta1 "github.com/rancher/opni/apis/grafana/v1beta1"
)

func init() {
	//addSchemeBuilders(opnigrafanav1alpha1.AddToScheme)
	addSchemeBuilders(opnigrafanav1beta1.AddToScheme)
}
