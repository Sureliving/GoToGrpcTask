package us.abstracta.jmeter.javadsl.sample;

import static org.assertj.core.api.Assertions.assertThat;
import static us.abstracta.jmeter.javadsl.JmeterDsl.*;

import java.io.IOException;
import java.time.Duration;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import us.abstracta.jmeter.javadsl.core.TestPlanStats;

public class PerformanceTest {

    @BeforeEach
    public void start() throws IOException, InterruptedException {
        int vu = 10;
        System.setProperty("VU", String.valueOf(vu));
        for (int i = 1; i <= vu; i++) {
            int port = 8090 + i;

            new ProcessBuilder("docker", "run", "-d",
                    "-p", port + ":" + port,
                    "--name", "greeter_client_" + port,
                    "-e", "PORT=" + port,
                    "gotogrpc/greeter_client:1.0.3").start();
        }
        Thread.sleep(10000);
    }

    @AfterEach
    public void finish() throws IOException, InterruptedException {
        Thread.sleep(10000);
        int vu = Integer.parseInt(System.getProperty("VU"));
        for (int i = 0; i <= vu; i++) {
            int port = 8090 + i;
            new ProcessBuilder("docker", "rm", "-f", "greeter_client_" + port).start();
        }
    }

    @Test
    public void testPerformance() throws IOException, InterruptedException {
        TestPlanStats searchMax = testPlan(
                rpsThreadGroup()
                        .maxThreads(500)
                        .rampToAndHold(50, Duration.ofSeconds(30), Duration.ofSeconds(60))
                        .rampToAndHold(100, Duration.ofSeconds(30), Duration.ofSeconds(60))
                        .rampToAndHold(150, Duration.ofSeconds(30), Duration.ofSeconds(60))
                        .rampToAndHold(100, Duration.ofSeconds(10), Duration.ofSeconds(30))
                        .children(
                                jsr223PreProcessor("import java.lang.Math\n"
                                        + "Integer vu = (props.getProperty('VU')).toInteger()\n"
                                        + "Integer port = 8090 + Math.abs(new Random().nextInt() % vu) + 1\n"
                                        + "vars.put('port', Integer.toString(port))"),
                                httpSampler("http://localhost:${port}/greet?name=${port}")
                        )
        ).run();
        assertThat(searchMax.overall().errorsCount()).isLessThan(10);
        Thread.sleep(10000);
        TestPlanStats stability = testPlan(
                rpsThreadGroup()
                        .maxThreads(500)
                        .rampToAndHold(100, Duration.ofSeconds(30), Duration.ofSeconds(600))
                        .children(
                                jsr223PreProcessor("import java.lang.Math\n"
                                        + "Integer vu = (props.getProperty('VU')).toInteger()\n"
                                        + "Integer port = 8090 + Math.abs(new Random().nextInt() % vu) + 1\n"
                                        + "vars.put('port', Integer.toString(port))"),
                                httpSampler("http://localhost:${port}/greet?name=${port}")
                        )
        ).run();
        assertThat(stability.overall().errorsCount()).isLessThan(10);
    }
}