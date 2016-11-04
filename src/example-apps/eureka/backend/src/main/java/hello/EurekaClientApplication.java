package hello;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.cloud.client.ServiceInstance;
import org.springframework.cloud.client.discovery.DiscoveryClient;
import org.springframework.cloud.client.discovery.EnableDiscoveryClient;
import org.springframework.stereotype.Component;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@EnableDiscoveryClient
@SpringBootApplication
public class EurekaClientApplication {

    public static void main(String[] args) {
        SpringApplication.run(EurekaClientApplication.class, args);
    }
}

@RestController
class ServiceInstanceRestController {

    private WhoAmI whoAmI;

    private DiscoveryClient discoveryClient;

    @Autowired
    public ServiceInstanceRestController(WhoAmI whoAmI, DiscoveryClient discoveryClient) {
        this.whoAmI = whoAmI;
        this.discoveryClient = discoveryClient;
    }

    @RequestMapping("/")
    public String index() {
        return
                "<ul>" +
                   "<li><a href=\"/whoami\">whoami</a>" +
                   "<li><a href=\"/instances\">instances</a>" +
                "</ul>";
    }

    @RequestMapping("/instances")
    public List<ServiceInstance> clients() {
        return this.discoveryClient.getInstances(whoAmI.springApplicationName);
    }

    @RequestMapping("/whoami")
    public WhoAmI whoami() {
        return whoAmI;
    }
}

@Component
class WhoAmI {
    @Value("${spring.application.name}")
    public String springApplicationName;

    @Value("${server.port:8080}")
    public String serverPort;
}