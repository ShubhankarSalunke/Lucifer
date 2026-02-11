from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from uuid import uuid4
from storage import (
    register_agent,
    update_agent_last_seen,
    create_experiment,
    get_experiment_for_agent,
    update_experiment_status,
    get_all_agents,
    get_all_experiments
)

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)
class AgentRegister(BaseModel):
    agent_id: str
    host: str


class ExperimentCreate(BaseModel):
    type: str
    target_container: str
    duration: int
    agent_id: str
    memory_mb: int | None = None


class ExperimentResult(BaseModel):
    experiment_id: str
    status: str
    result: dict | None = None


@app.post("/register")
def register(agent: AgentRegister):
    register_agent(agent.agent_id, agent.host)
    return {"message": "Agent registered"}


@app.get("/poll/{agent_id}")
def poll(agent_id: str):
    update_agent_last_seen(agent_id)
    exp = get_experiment_for_agent(agent_id)
    return exp if exp else {}



@app.post("/result")
def submit_result(result: ExperimentResult):
    update_experiment_status(
        result.experiment_id,
        result.status,
        result.result
    )
    return {"message": "Result recorded"}

@app.post("/create-experiment")
def create_exp(exp: ExperimentCreate):
    exp_id = str(uuid4())

    create_experiment(exp_id, exp.dict())

    return {"experiment_id": exp_id}


@app.get("/agents")
def agents():
    return get_all_agents()


@app.get("/experiments")
def experiments():
    return get_all_experiments()
