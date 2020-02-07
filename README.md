# Cadence Samples
These are some samples to demostrate various capabilities of Cadence client and server.  You can learn more about cadence at:
* Cadence: https://github.com/uber/cadence
* Cadence Client: https://github.com/uber-go/cadence-client

## Prerequisite
Run Cadence Server

See instructions for running the Cadence Server: https://github.com/uber/cadence/blob/master/README.md

See instructions for using CLI to register a domain(name as "samples-domain"): https://cadenceworkflow.io/docs/08_cli
 or https://github.com/uber/cadence/blob/master/tools/cli/README.md 
 
 
## Steps to run samples
### Build Samples
```
make
```

### Run HelloWorld Sample
* Start workers for helloworld workflow and activities
```
./bin/helloworld -m worker
```
* Start workflow execution for helloworld workflow
```
./bin/helloworld -m trigger
```

### Commands to run other samples

#### cron
```
./bin/cron -m worker
```
Start workflow with cron expression scheduled to run every minute.
```
./bin/cron -m trigger -cron "* * * * *"
```

#### dsl
```
./bin/dsl -m worker
```
```
./bin/dsl -m trigger -dslConfig cmd/samples/dsl/workflow1.yaml
./bin/dsl -m trigger -dslConfig cmd/samples/dsl/workflow2.yaml
```

#### expense
See more details in https://github.com/zokypesch/cadence-samples/blob/master/cmd/samples/expense/README.md

#### fileprocessing
```
./bin/fileprocessing -m worker
```
```
./bin/fileprocessing -m trigger
```

#### recipes/branch
```
./bin/branch -m worker
```
Run branch workflow
```
./bin/branch -m trigger -c branch
```
Run parallel branch workflow
```
./bin/branch -m trigger -c parallel this will run the parallel branch workflow
```

#### recipes/choice
```
./bin/choice -m worker
```
Run the single choice workflow
```
./bin/choice -m trigger -c single
```
Run the multi choice workflow
```
./bin/choice -m trigger -c multi
```

#### greetings
```
./bin/greetings -m worker
```
```
./bin/greetings -m trigger
```

#### pickfirst
```
./bin/pickfirst -m worker
```
```
./bin/pickfirst -m trigger
```

#### mutex
```
./bin/mutex -m worker
```
```
./bin/mutex -m trigger
```

#### retryactivity
```
./bin/retryactivity -m worker
```
```
./bin/retryactivity -m trigger
```

#### splitmerge
```
./bin/splitmerge -m worker
```
```
./bin/splitmerge -m trigger
```

#### timer
```
./bin/timer -m worker
```
```
./bin/timer -m trigger
```

#### childworkflow
```
./bin/childworkflow -m worker
```
```
./bin/childworkflow -m trigger
```

#### dynamic
```
./bin/dynamic -m worker
```
```
./bin/dynamic -m trigger
```

#### localactivity
See more details in https://github.com/zokypesch/cadence-samples/blob/master/cmd/samples/recipes/localactivity/README.md

#### query
See more details in https://github.com/zokypesch/cadence-samples/blob/master/cmd/samples/recipes/query/README.md

#### recovery
See more details in https://github.com/zokypesch/cadence-samples/blob/master/cmd/samples/recovery/README.md

for replace string
:Ack -l 'zokypesch' | xargs perl -pi -E 's/zokypesch/zokypesch/g'



