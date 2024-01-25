#!/usr/bin/env -S locust -f
# https://docs.locust.io/en/stable/quickstart.html

from locust import HttpUser, task, tag, events, stats

stats.PERCENTILES_TO_REPORT = [0.25, 0.50, 0.75, 0.80, 0.90, 0.95, 0.98, 0.99, 1.0]

@events.init_command_line_parser.add_listener
def arguments(parser):
    parser.add_argument("--tsp-n", type=int, env_var="TSP_N", default=10, help="random set size for travelling_salesman")

class ServerledgeWorkload(HttpUser):

    # default url prefix of broker to load
    host = "http://broker.ansemjo.de:1323"

    def __invoke(self, function, params, name=None):
        "Helper function to invoke a serverledge function."
        return self.client.post(
            f"/invoke/{function}",
            json={
                "Async": False,
                "CanDoOffloading": True,
                "QoSClass": 0,
                "QoSMaxRespT": -1,
                "Params": params,
            },
            name=name if name is not None else f"invoke/{function}",
            catch_response=True, # need to look into response (TODO: do we?)
        )

    def invoke(self, function, params, name=None):
        "Catch common errors that might happen during an invocation."
        with self.__invoke(function, params, name) as response:
            if not response.ok:
                print(f"task not OK: {response.status_code} {response.reason}")
                return response.failure("not successful")
                # raise exception.RescheduleTask()
            if not '"Success":true' in response.text:
                print(f"task not successful: {response.text}")
                return response.failure("not successful")


    @task(1)
    @tag("tsp")
    def task_tsp_rand(self):
        "Travelling Salesman Problem with `n` random cities."
        n = str(self.environment.parsed_options.tsp_n)
        self.invoke("tsp", dict(n=n), f"invoke/tsp({n})")

    # @task()
    # @tag("isprime")
    # def task_example_isprime(self):
    #     "The isprime.py example from Serverledge repo."
    #     self.invoke("isprime", dict(n=42))