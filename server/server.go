package server

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	// EKS requires the AWS SDK to be imported

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"

	// "honnef.co/go/tools/conf
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	// pb "github.com/example/mypackage"

	"github.com/clusterService/server/handler"
	models "github.com/clusterService/server/model"
	services "github.com/clusterService/server/services"
	utils "github.com/clusterService/server/utils"
)

// Server is srv struct that holds srv Kubernetes client
type Server struct {
	KubeClient *kubernetes.Clientset
}

// const ETCDIP = "34.198.214.90:30000--"
const ETCDIP = "192.168.59.100:30000"

func ApiMiddleware(cli *kubernetes.Clientset) gin.HandlerFunc {
	// do something with the request
	return func(c *gin.Context) {
		// do something with the request

		c.Set("kubeClient", cli)
		c.Next()
	}
}

func (srv *Server) Initialize() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	srv = &Server{KubeClient: client}
	ctx := context.Background()

	//listing the namespaces
	namespaces, _ := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	for _, namespace := range namespaces.Items {
		fmt.Println(namespace.Name)
	}
	// initPods := initPodCreation(client)
	initPods := initSetOfPodCreation(client)

	fmt.Println(initPods)

	fmt.Println("=================================starting server=================================")
	r := gin.Default()
	r.Use(ApiMiddleware(client))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	/** temop routes start */
	r.POST("/CreateDeployment", handler.CreateDeployment)
	r.POST("/UpdateDeployment", handler.UpdateDeployment)
	r.GET("/getDeployment", handler.GetDeployment)
	r.GET("/getAllDeployments", handler.GetAllDeployments)
	r.DELETE("/deleteDeployment", handler.DeleteDeployment)

	r.POST("/CreateService", handler.CreateService)
	r.GET("/getService", handler.GetService)
	r.GET("/getAllServices", handler.GetAllServices)
	r.DELETE("/deleteService", handler.DeleteService)

	r.POST("/CreatePod", handler.CreatePod)
	r.GET("/getPod", handler.GetPod)
	r.GET("/getAllPods", handler.GetAllPods)
	r.DELETE("/deletePod", handler.DeletePod)

	r.Run(":8081")
}

func initPodCreation(cli *kubernetes.Clientset) string {
	c := &gin.Context{}

	c.Set("kubeClient", cli)
	fmt.Println("Creating Pods")
	for _, image := range utils.ImagesList {
		fmt.Println("Creating Deployment for %s", image)
		deployment, err := services.CreateDeployment(c, &models.CreateDeploymentRequest{
			Name:     image.FuncName,
			Replicas: 1,
			Selector: map[string]string{
				"app": image.FuncName,
			},
			Labels: map[string]string{
				"app": image.FuncName,
			},
			Containers: []corev1.Container{
				{
					Name:  image.FuncName + "-container",
					Image: image.ImageName,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: image.Port,
						},
					},
					////ImagePullPolicy: corev1.PullNever,
				},
			},
		})
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(deployment)
		fmt.Println("=================================")
		fmt.Println("deployment done")
		// strings.ReplaceAll(strings.ReplaceAll(image, ":", "-"), "/", "-")
		service, err := services.CreateService(c, &models.CreateServiceRequest{
			Name: image.FuncName,
			Selector: map[string]string{
				"app": image.FuncName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:     image.Port,
					NodePort: image.NodePort,
				},
			},
			Type: "NodePort",
		})
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(service)

	}
	return "init Pods Created Successfully"
}

func initSetOfPodCreation(cli *kubernetes.Clientset) string {
	c := &gin.Context{}

	c.Set("kubeClient", cli)
	fmt.Println("Creating Pods")
	for _, image := range utils.PbftImagesList {

		fmt.Println("Creating set of pods and registering them with etcd for %s", image)
		fmt.Println("Creating:", image.ImageName+"-worker:latest")

		// create worker pod for each operation image
		// createPodRequest := models.CreatePodRequest{
		// 	Name: image.FuncName + "-worker",
		// 	Containers: []corev1.Container{
		// 		{
		// 			Name:  image.FuncName + "-worker",
		// 			Image: image.ImageName + "-worker:latest",
		// 			Ports: []corev1.ContainerPort{
		// 				{
		// 					ContainerPort: image.Port,
		// 				},
		// 			},
		// 			// ////ImagePullPolicy: corev1.PullNever,
		// 		},
		// 	},
		// }
		// createWorkerPodResponse, er := services.CreatePod(c, &createPodRequest)
		deployment, err := services.CreateDeployment(c, &models.CreateDeploymentRequest{
			Name:     image.FuncName + "-worker",
			Replicas: 1,
			Selector: map[string]string{
				"app": image.FuncName + "-worker",
			},
			Labels: map[string]string{
				"app": image.FuncName + "-worker",
			},
			Containers: []corev1.Container{
				{
					Name:  image.FuncName + "-worker",
					Image: image.ImageName + "-worker:latest",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: image.Port,
						},
					},
					// ImagePullPolicy: corev1.PullNever,
				},
			},
		})
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(deployment)
		fmt.Println("=================================")
		fmt.Println("deployment done")
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("pod creation done")

		fmt.Println("================================================")
		// create a service which exposes the worker pod via load balancer

		service, err := services.CreateService(c, &models.CreateServiceRequest{
			Name: image.FuncName + "-worker",
			Selector: map[string]string{
				"app": image.FuncName + "-worker",
			},
			Ports: []corev1.ServicePort{
				{
					Port:     image.Port,
					NodePort: image.NodePort,
				},
			},
			Type: "NodePort",
		})

		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("================================================================worker service creation done================================================================")
		fmt.Println(service)

		//  Create an etcd client
		// etcdClient, err := clientv3.New(clientv3.Config{
		// 	Endpoints:   []string{ETCDIP},
		// 	DialTimeout: 5 * time.Second,
		// })
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// defer etcdClient.Close()

		// // Register the pod with etcd
		// key := fmt.Sprintf("pods/%s", "set-"+strconv.Itoa(int(image.Id))+"_"+image.FuncName+"-worker")
		// value := fmt.Sprintf(":%s", ":"+strconv.Itoa(int(image.Port)))
		// _, err = etcdClient.Put(context.Background(), key, value)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// fmt.Printf("Registered pod with etcd: %s -> %s\n", key, value)

		// for loop of to iterate over the 4 pods an deploye the replicas and register them
		for i := 0; i < 5; i++ {
			fmt.Println("==========================================started creating replicas for each id=================================================================")
			fmt.Println(i)
			// create replica pod for each id
			// createPodRequest := models.CreatePodRequest{
			// 	Name: image.FuncName + "-replica-" + strconv.Itoa(i),
			// 	Containers: []corev1.Container{
			// 		{
			// 			Name:  image.FuncName + "-replica-" + strconv.Itoa(i),
			// 			Image: image.ImageName + "-replica:latest",
			// 			Ports: []corev1.ContainerPort{
			// 				{
			// 					ContainerPort: image.Port,
			// 				},
			// 			},
			// 			// ////ImagePullPolicy: corev1.PullNever,
			// 		},
			// 	},
			// }
			deployment, err := services.CreateDeployment(c, &models.CreateDeploymentRequest{
				Name:     image.FuncName + "-replica-" + strconv.Itoa(i),
				Replicas: 1,
				Selector: map[string]string{
					"app": image.FuncName + "-replica-" + strconv.Itoa(i),
				},
				Labels: map[string]string{
					"app": image.FuncName + "-replica-" + strconv.Itoa(i),
				},
				Containers: []corev1.Container{
					{
						Name:  image.FuncName + "-replica-" + strconv.Itoa(i),
						Image: image.ImageName + "-replica:latest",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: image.Port,
							},
						},
						// ImagePullPolicy: corev1.PullNever,
					},
				},
			})

			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("replica pod deployment done")

			fmt.Println(deployment)
			fmt.Println("===========================registering with etcd=====================")
			//  Create an etcd client
			// etcdClient, err := clientv3.New(clientv3.Config{
			// 	Endpoints:   []string{ETCDIP},
			// 	DialTimeout: 5 * time.Second,
			// })
			// if err != nil {
			// 	log.Fatal(err)
			// }
			// defer etcdClient.Close()
			// // Register the pod with etcd
			// key := fmt.Sprintf("pods/%s", "set-"+strconv.Itoa(int(image.Id))+"_"+image.FuncName+"-replica-"+strconv.Itoa(i))
			// value := fmt.Sprintf(":%s", ":"+strconv.Itoa(int(image.Port)))
			// _, err = etcdClient.Put(context.Background(), key, value)
			// if err != nil {
			// 	log.Fatal(err)
			// }
			// fmt.Printf("Registered pod with etcd: %s -> %s\n", key, value)
		}

	}

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{ETCDIP},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer etcdClient.Close()

	fmt.Println("Getting the etcd values ***********************")
	// Perform the Range operation without specifying a key range
	resp, err := etcdClient.Get(context.Background(), "", clientv3.WithPrefix())
	if err != nil {
		log.Fatal(err)
	}

	// Check if any value exists
	if len(resp.Kvs) == 0 {
		fmt.Println("No values found")
	}

	// Access the stored values
	for _, kv := range resp.Kvs {
		fmt.Printf("Key *************: %s, Value***********: %s\n", kv.Key, kv.Value)
	}
	// strings.ReplaceAll(strings.ReplaceAll(image, ":", "-"), "/", "-")
	// service, err := services.CreateService(c, &models.CreateServiceRequest{
	// 	Name: image.FuncName,
	// 	Selector: map[string]string{
	// 		"app": image.FuncName,
	// 	},
	// 	Ports: []corev1.ServicePort{
	// 		{
	// 			Port:     image.Port,
	// 			NodePort: image.NodePort,
	// 		},
	// 	},
	// 	Type: "NodePort",
	// })
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(service)

	// fmt.Println("Creating set of pods and registering them with etcd for %s", "rajeshreddyt/grpcservermultiply:latest")
	// fmt.Println("Creating:", "rajeshreddyt/grpcservermultiply:latest"+"-worker:latest")

	// // create worker pod for each operation image
	// createPodRequest := models.CreatePodRequest{
	// 	Name: "multiply-worker",
	// 	Containers: []corev1.Container{
	// 		{
	// 			Name:  "multiply-worker",
	// 			Image: "rajeshreddyt/grpcservermultiply:latest",
	// 			Ports: []corev1.ContainerPort{
	// 				{
	// 					ContainerPort: 3000,
	// 				},
	// 			},
	// 			// ////ImagePullPolicy: corev1.PullNever,
	// 		},
	// 	},
	// }
	// createWorkerPodResponse, er := services.CreatePod(c, &createPodRequest)
	// if er != nil {
	// 	fmt.Println(er)
	// }
	// fmt.Println("pod creation done")
	// fmt.Println(createWorkerPodResponse)
	// fmt.Println("================================================")
	// // create a service which exposes the worker pod via nodeport
	// createServiceRequest := models.CreateServiceRequest{
	// 	Name: "multiply-worker",
	// 	Selector: map[string]string{
	// 		"app": "multiply-worker",
	// 	},
	// 	Ports: []corev1.ServicePort{
	// 		{
	// 			Port:     3000,
	// 			Protocol: corev1.ProtocolTCP,
	// 			NodePort: 30003,
	// 		},
	// 	},
	// 	Type: "NodePort",
	// }
	// createServiceResponse, er := services.CreateService(c, &createServiceRequest)
	// if er != nil {
	// 	fmt.Println(er)
	// }
	// fmt.Println("================================================================service creation done================================================================")
	// fmt.Println(createServiceResponse)
	return "init Pods Created Successfully"
}

func truncateAfterWord(str, word string) string {
	index := strings.Index(str, word)
	if index != -1 {
		return str[:index+len(word)]
	}
	return str
}
