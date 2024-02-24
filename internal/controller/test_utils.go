package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	angiv1 "ryan.evans.com/angi/api/v1"
)

func getPodInfoCustomResourceDefinition(resourceName string, memoryLimit string, cupRequest string,
	imageRepo string, imageTag string, uiColor string, uiMessage string, redisEnabled bool) *angiv1.PodInfo {

	return &angiv1.PodInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: "default",
		},
		Spec: angiv1.PodInfoSpec{
			ReplicaCount: 2,
			Resources: angiv1.ResourceInfo{
				MemoryLimit: memoryLimit,
				CpuRequest:  cupRequest,
			},
			Image: angiv1.ImageInfo{
				Repository: imageRepo,
				Tag:        imageTag,
			},
			UI: angiv1.UIInfo{
				Color:   uiColor,
				Message: uiMessage,
			},
			Redis: angiv1.RedisInfo{
				Enabled: redisEnabled,
			},
		},
	}
}
