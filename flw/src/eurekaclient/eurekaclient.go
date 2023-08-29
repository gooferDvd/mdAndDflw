package eurekaclient

import eureka "github.com/xuanbo/eureka-client"

func EurekaClient(eurekazone string, port int) {

	client := eureka.NewClient(&eureka.Config{
		DefaultZone:           eurekazone,
		App:                   "docker-flow",
		Port:                  port,
		RenewalIntervalInSecs: 10,
		DurationInSecs:        30,
		Metadata: map[string]interface{}{
			"VERSION":              "0.0.1",
			"NODE_GROUP_ID":        0,
			"PRODUCT_CODE":         "DEFAULT",
			"PRODUCT_VERSION_CODE": "DEFAULT",
			"PRODUCT_ENV_CODE":     "DEFAULT",
			"SERVICE_VERSION_CODE": "DEFAULT",
		},
	})
	client.Start()
}