# Project 1: Process Scheduler

## Description 
For this project we'll be building a simple process scheduler that takes in a file containing example processes, and outputs a schedule based on the three different schedule types:

- First Come First Serve (FCFS) [already done]
- Shortest Job First (SJF)
- SJF Priority
- Round-robin (RR)
- Assume that all processes are CPU bound (they do not block for I/O).

The scheduler will be written in [Go](https://go.dev/) (a skeleton main.go is included in the project repo).

## Steps

1. Clone down the example input/output and skeleton main.go:

   1. `git clone https://github.com/jh125486/CSCE4600`

2. Copy the `Project1` files to your own git project.
    
    1. In your `go.mod`, replace "jh125486" in the module line with your GitHub name, e.g.:

      - "module github.com/jh125486/CSCE4600" changes to "module github.com/CoolStudent123/CSCE4600"

    2. In the `main.go`, replace "jh125486" in the imports with your package path, e.g.:

      - "github.com/jh125486/CSCE4600/Project2/builtins" changes to "github.com/CoolStudent123/CSCE4600/Project2/builtins"

4. The processes for your scheduling algorithms are read from a file as the first argument to your program.

   1. Every line in this file includes a record with comma separated fields.

      1. The format for this record is the following: \<ProcessID>,\<Burst Duration>,\<Arrival Time>,\<Priority>.

   2. Not all fields are used by all scheduling algorithms. For example, for FCFS you only need the process IDs, arrival times, and burst durations.

   3. All processes in your input files will be provided a unique process ID. The arrival times and burst durations are integers. Process priorities have a range of [1-50]; the lower this number, the higher the priority i.e. a process with priority=1 has a higher priority than a process with priority=2.

5. Start editing the `main.go` and add the scheduling algorithms:
   1. Implement SJF (preemptive) and report average turnaround time, average waiting time, and average throughput.

   2. Implement SJF priority scheduling (preemptive) and report average turnaround time, average waiting time, and average throughput.

   3. Round-round (preemptive) and report average turnaround time, average waiting time, and average throughput.


## Deliverables

A GitHub link to your project which includes:

- `README.md` <- describes anything needed to build (optional)
- `main.go` <- your scheduler
