# Monitor E2E Interaction Flow

```mermaid
flowchart TD
  Start["Monitor.Start(ctx)"] --> InitSched["scheduler.Initialize(ctx)"]
  InitSched --> NodeSync["logNodeSyncStatus(ctx)"]
  NodeSync --> InitRecon["reconciler.InitializeCursors(ctx)"]
  InitRecon --> StartWorkers["workerPool.Start(ctx)"]
  StartWorkers --> StartResultLoop[spawn processResults loop]
  StartWorkers --> StartRealtimeLoop[spawn realtime.Run loop]
  StartWorkers --> StartReconLoop[spawn reconcile.Run loop]

  subgraph realtimePath [Realtime Path]
    RTWait[realtime waitForInterval] --> RTHead["headSlotCache.Get(ctx)"]
    RTHead --> RTEvents["scheduler.NextEvents(slot)"]
    RTEvents --> RTSwitch{event type}
    RTSwitch -->|SlotPoll| D1[dispatcher.PollValidatorsForSlotEpoch]
    RTSwitch -->|EpochBoundary| D2[dispatcher.FetchDutiesForEpoch]
    RTSwitch -->|EpochFinalized| D3[dispatcher.FetchRewardsForEpoch]
    D1 --> Enqueue
    D2 --> Enqueue
    D3 --> Enqueue
  end

  subgraph reconPath [Reconcile Path]
    RLoop[reconcile.Run loop] --> RHead["headSlotCache.Get(ctx)"]
    RHead --> RSnap[reconcileSnapshots bounded catch-up]
    RHead --> REpoch[reconcileEpochData bounded catch-up]
    RSnap --> RDispatch1[dispatcher.PollValidatorsForSlotEpoch]
    REpoch --> RDispatch2[dispatcher.FetchDutiesForEpoch]
    REpoch --> RDispatch3[dispatcher.FetchRewardsForEpoch]
    RDispatch1 --> Enqueue
    RDispatch2 --> Enqueue
    RDispatch3 --> Enqueue
  end

  Enqueue["workerPool.Submit(ctx, job)"] --> WorkerN[Worker goroutines]
  WorkerN --> Process["jobs.Processor.Process"]
  Process --> StatusH["StatusHandler.Process"]
  Process --> DutiesH["DutiesHandler.Process"]
  Process --> RewardsH["RewardsHandler.Process"]

  StatusH --> Repo1[repo.SaveValidatorSnapshot]
  DutiesH --> Repo2[repo.SaveAttestationDuties]
  RewardsH --> Repo3[repo.SaveAttestationRewards + penalties]
  Repo1 --> ResultChan
  Repo2 --> ResultChan
  Repo3 --> ResultChan

  ResultChan["workerPool.Results()"] --> ResultsLoop[processResults loop]
  ResultsLoop --> HandleResult[handleResult]
  HandleResult --> LogOK[logStatus / logDuties / logRewards]
  HandleResult --> LogErr[job error logging]

  Stop["Monitor.Stop()"] --> CloseJobs[workerPool.Stop close jobChan]
  CloseJobs --> WaitWorkers[wait worker goroutines]
  WaitWorkers --> CloseResults[close resultChan]
  CloseResults --> WaitAll["monitor wg.Wait"]
  WaitAll --> endNode[Monitor stopped]
```

## Realtime monitoring (single linear flow)

End-to-end path for the **realtime** loop only: pacing → head → schedule → enqueue → work → log.

```mermaid
flowchart LR
  A["Start: realtime.Runner.Run"] --> B["Wait: waitForInterval"]
  B --> C["Head: getHead"]
  C --> D["Plan: handleForSlot, NextEvents"]
  D --> E["Dispatch: dispatcher"]
  E --> F["Enqueue: workerPool.Submit"]
  F --> G["Execute: jobs.Processor"]
  G --> H["Persist: beacon + repo"]
  H --> I["Emit: core.Result"]
  I --> J["Log: processResults"]
  J --> B
```

**In one sentence:** the realtime goroutine wakes on each interval, reads the head slot, turns that slot into scheduled events and dispatcher calls, workers fetch and persist, results are emitted and logged—then the cycle repeats until `ctx` is done.

## Module and package call graph

This view focuses on **which package calls which package**.

```mermaid
flowchart LR
  M["internal/monitor (Monitor)"] --> C["internal/monitor/cache"]
  M --> S["internal/monitor/scheduler"]
  M --> D["internal/monitor/dispatch"]
  M --> W["internal/monitor (worker pool)"]
  M --> RT["internal/monitor/runners/realtime"]
  M --> RC["internal/monitor/runners/reconcile"]
  M --> R["internal/monitor/results"]

  RT --> C
  RT --> D
  RC --> C
  RC --> D

  D --> Core["internal/monitor/core (Job, Result)"]
  D --> W

  W --> J["internal/monitor/jobs (Processor)"]
  J --> H1["jobs/status handler"]
  J --> H2["jobs/duties handler"]
  J --> H3["jobs/rewards handler"]

  H1 --> B["internal/beacon"]
  H2 --> B
  H3 --> B

  H1 --> Store["internal/store interfaces"]
  H2 --> Store
  H3 --> Store
  Store --> PG["internal/storage/postgres"]

  W --> R
  R --> Core
```

## Startup

1. `Monitor.Start(ctx)` initializes scheduler, logs node sync status, and initializes reconciliation cursors.
2. Starts worker pool.
3. Spawns three long-running goroutines:
   - realtime runner
   - reconcile runner
   - result processor

## Realtime Path

1. Realtime runner waits for interval.
2. Reads current slot from head-slot cache.
3. Asks scheduler for events at current slot.
4. Dispatches events into jobs through dispatcher.
5. Jobs are enqueued into worker pool.

## Reconcile Path

1. Reconcile runner reads head slot from cache.
2. Runs bounded catch-up for snapshots and epoch data.
3. Uses dispatcher to enqueue corresponding jobs.
4. Repeats until caught up (or context canceled).

## Job Execution Path

1. Workers consume jobs from queue.
2. `jobs.Processor` routes to job-specific handler:
   - status
   - duties
   - rewards
3. Handler fetches beacon data and writes to repository.

## Result Processing Path

1. Worker emits `core.Result` to result channel.
2. `processResults` consumes results.
3. Logs success payloads or failure details.

## Shutdown

1. `Monitor.Stop()` calls `workerPool.Stop()`.
2. Job channel closes, workers drain and exit.
3. Result channel closes.
4. `wg.Wait()` waits for all monitor goroutines to finish.
5. Monitor reports stopped.
