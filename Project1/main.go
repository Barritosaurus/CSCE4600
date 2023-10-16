package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func main() {
	// CLI args
	f, closeFile, err := openProcessingFile(os.Args...)
	if err != nil {
		log.Fatal(err)
	}
	defer closeFile()

	// Load and parse processes
	processes, err := loadProcesses(f)
	if err != nil {
		log.Fatal(err)
	}

	// First-come, first-serve scheduling
	FCFSSchedule(os.Stdout, "First-come, first-serve", processes)
	SJFSchedule(os.Stdout, "Shortest-job-first", processes)
	SJFPrioritySchedule(os.Stdout, "Priority", processes)
	RRSchedule(os.Stdout, "Round-robin", processes)
}

func openProcessingFile(args ...string) (*os.File, func(), error) {
	if len(args) != 2 {
		return nil, nil, fmt.Errorf("%w: must give a scheduling file to process", ErrInvalidArgs)
	}
	// Read in CSV process CSV file
	f, err := os.Open(args[1])
	if err != nil {
		return nil, nil, fmt.Errorf("%v: error opening scheduling file", err)
	}
	closeFn := func() {
		if err := f.Close(); err != nil {
			log.Fatalf("%v: error closing scheduling file", err)
		}
	}

	return f, closeFn, nil
}

type (
	Process struct {
		ProcessID     int64
		ArrivalTime   int64
		BurstDuration int64
		Priority      int64
	}
	TimeSlice struct {
		PID   int64
		Start int64
		Stop  int64
	}
)

//region Schedulers

// FCFSSchedule outputs a schedule of processes in a GANTT chart and a table of timing given:
// • an output writer
// • a title for the chart
// • a slice of processes
func FCFSSchedule(w io.Writer, title string, processes []Process) {
	var (
		serviceTime     int64
		time       float64
		totalTurnaround float64
		lastCompletion  float64
		waitingTime     int64
		schedule        = make([][]string, len(processes))
		gantt           = make([]TimeSlice, 0)
	)
	for i := range processes {
		if processes[i].ArrivalTime > 0 {
			waitingTime = serviceTime - processes[i].ArrivalTime
		}
		time += float64(waitingTime)

		start := waitingTime + processes[i].ArrivalTime

		turnaround := processes[i].BurstDuration + waitingTime
		totalTurnaround += float64(turnaround)

		completion := processes[i].BurstDuration + processes[i].ArrivalTime + waitingTime
		lastCompletion = float64(completion)

		schedule[i] = []string{
			fmt.Sprint(processes[i].ProcessID),
			fmt.Sprint(processes[i].Priority),
			fmt.Sprint(processes[i].BurstDuration),
			fmt.Sprint(processes[i].ArrivalTime),
			fmt.Sprint(waitingTime),
			fmt.Sprint(turnaround),
			fmt.Sprint(completion),
		}
		serviceTime += processes[i].BurstDuration

		gantt = append(gantt, TimeSlice{
			PID:   processes[i].ProcessID,
			Start: start,
			Stop:  serviceTime,
		})
	}

	count := float64(len(processes))
	aveWait := time / count
	aveTurnaround := totalTurnaround / count
	aveThroughput := count / lastCompletion

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

func SJFSchedule(w io.Writer, title string, processes []Process) {
	var (
		total 			int = 0
		min 			int64 = math.MaxInt64
		shortest 		int64 = 0
		time       		int64
		totalTurnaround int64
		totalWait		int64
		check 			bool = false
		schedule        = make([][]string, len(processes))
		gantt           = make([]TimeSlice, 0)
		recordedTimes 	= make([]int64, len(processes))
		waitTimes		= make([]int64, len(processes))
		turnArounds		= make([]int64, len(processes))
		completions		= make([]int64, len(processes))
	)

	// copy burst durations for tracking
	for i := range processes {
		recordedTimes[i] = processes[i].BurstDuration
	}

	// run until all processes are complete
	for total != len(processes) {
		
		// find process with minimum remaining time
		for i := range processes {
			if processes[i].ArrivalTime <= time && (recordedTimes[i] < min) && recordedTimes[i] > 0 {
				min = recordedTimes[i]
				shortest = int64(i)
				check = true
			}
		}

		// if no process is ready
		if check == false {
			time++
			continue
		}

		// reduce remaining time
		recordedTimes[shortest]--

		// update minimum
		min = recordedTimes[shortest]
		if(min == 0){
			min = math.MaxInt64
		}

		// if fully executed
		if recordedTimes[shortest] == 0 {
			total++
			check = false
			waitTimes[shortest] = time - processes[shortest].BurstDuration - processes[shortest].ArrivalTime
			completions[shortest] =  processes[shortest].BurstDuration + processes[shortest].ArrivalTime + waitTimes[shortest]

			if waitTimes[shortest] < 0 {
				waitTimes[shortest] = 0
			}
		}

		time++
	}

	// calculate turnarounds
	for i := range processes {
		turnArounds[i] = processes[i].BurstDuration + waitTimes[i]
	}

	// provide output schedule
	for i := range processes {
		schedule[i] = []string{
			fmt.Sprint(processes[i].ProcessID),
			fmt.Sprint(processes[i].Priority),
			fmt.Sprint(processes[i].BurstDuration),
			fmt.Sprint(processes[i].ArrivalTime),
			fmt.Sprint(waitTimes[i]),
			fmt.Sprint(turnArounds[i]),
			fmt.Sprint(completions[i]),
		}

		gantt = append(gantt, TimeSlice{
			PID:   processes[i].ProcessID,
			Start: completions[i] - waitTimes[i],
			Stop:  completions[i] - waitTimes[i] + processes[i].BurstDuration,
		})
	}

	// calculate averages
	for i := range processes {
		totalTurnaround += turnArounds[i]
	}
	for i := range processes {
		totalWait += waitTimes[i]
	}

	count := float64(len(processes))
	aveWait := float64(totalWait) / count
	aveTurnaround := float64(totalTurnaround) / count
	aveThroughput := count / float64(time)

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

func SJFPrioritySchedule(w io.Writer, title string, processes []Process) { 
	var (
		total 			int = 0
		min 			int64 = math.MaxInt64
		curr 			int64 = 0
		time       		int64
		totalTurnaround int64
		totalWait		int64
		check 			bool = false
		schedule        = make([][]string, len(processes))
		gantt           = make([]TimeSlice, 0)
		recordedTimes 	= make([]int64, len(processes))
		waitTimes		= make([]int64, len(processes))
		turnArounds		= make([]int64, len(processes))
		completions		= make([]int64, len(processes))
	)

	// copy burst durations for tracking
	for i := range processes {
		recordedTimes[i] = processes[i].BurstDuration
	}

	// run until all processes are complete
	for total != len(processes) {

		// find process with highest priority and minimum remaining time
		for i := range processes {
			if processes[i].ArrivalTime <= time && ((processes[i].Priority < processes[curr].Priority) || recordedTimes[i] < min) && recordedTimes[i] > 0 {
				min = recordedTimes[i]
					curr = int64(i)
				check = true
			}
		}

		// if no process is ready
		if check == false {
			time++
			continue
		}

		// reduce remaining time
		recordedTimes[curr]--

		// update minimum
		min = recordedTimes[curr]
		if(min == 0){
			min = math.MaxInt64
		}

		// if fully executed
		if recordedTimes[curr] == 0 {
			total++
			check = false
			waitTimes[curr] = time - processes[curr].BurstDuration - processes[curr].ArrivalTime
			completions[curr] =  processes[curr].BurstDuration + processes[curr].ArrivalTime + waitTimes[curr]

			if waitTimes[curr] < 0 {
				waitTimes[curr] = 0
			}
		}

		time++
	}

	// calculate turnarounds
	for i := range processes {
		turnArounds[i] = processes[i].BurstDuration + waitTimes[i]
	}

	// provide output schedule
	for i := range processes {
		schedule[i] = []string{
			fmt.Sprint(processes[i].ProcessID),
			fmt.Sprint(processes[i].Priority),
			fmt.Sprint(processes[i].BurstDuration),
			fmt.Sprint(processes[i].ArrivalTime),
			fmt.Sprint(waitTimes[i]),
			fmt.Sprint(turnArounds[i]),
			fmt.Sprint(completions[i]),
		}

		gantt = append(gantt, TimeSlice{
			PID:   processes[i].ProcessID,
			Start: completions[i] - waitTimes[i],
			Stop:  completions[i] - waitTimes[i] + processes[i].BurstDuration,
		})
	}

	// calculate averages
	for i := range processes {
		totalTurnaround += turnArounds[i]
	}
	for i := range processes {
		totalWait += waitTimes[i]
	}



	count := float64(len(processes))
	aveWait := float64(totalWait) / count
	aveTurnaround := float64(totalTurnaround) / count
	aveThroughput := count / float64(time)

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

func RRSchedule(w io.Writer, title string, processes []Process) {
	var (
		tq 					int64 = 2
		time       			int64 = processes[0].ArrivalTime
		totalTurnaround 	int64
		totalWait			int64
		highestIndex 		int = 0
		schedule        	= make([][]string, len(processes))
		gantt           	= make([]TimeSlice, 0)
		waitTimes			= make([]int64, len(processes))
		turnArounds			= make([]int64, len(processes))
		completions			= make([]int64, len(processes))
		recordedTimes		= make([]int64, len(processes))
		queue				= make([]int64, len(processes))
	)

	// prepare recordedTimes
	for i := range processes {
		recordedTimes[i] = processes[i].BurstDuration
	}

	for i := range processes {
		completions[i] = -1
		waitTimes[i] = -1
		turnArounds[i] = -1
		queue[i] = 0
	}
	queue[0] = 1

	for {
		var flag bool = true
		for i := range processes {
			if recordedTimes[i] != 0 {
				flag = false
				break
			}
		}
		if flag {
			break
		}

		for i := 0; (i < len(processes) && (queue[i] != 0)); i++ {
			var curr int64 = 0
			for (curr < tq) && (recordedTimes[queue[0]-1] > 0) {
				recordedTimes[queue[0]-1] -= 1
				time += 1
				curr++

				// check new arrival
				if time <= processes[len(processes) - 1].ArrivalTime {
					var newArrival bool = false
					for j := (highestIndex + 1); j < len(processes); j++ {
						if processes[j].ArrivalTime <= time {
							if highestIndex < j {
								highestIndex = j
								newArrival = true
							}
						}

					}

					// add new arrivals to the queue
					if newArrival {
						var index int = 0
						for j := range processes {
							if queue[j] == 0 {
								index = j
								break
							}
						}
						queue[index] = int64(highestIndex + 1)
					}
				}
			}

			// if process is complete then store its exit
			if (recordedTimes[queue[0]-1] == 0) && (completions[queue[0]-1] == -1) {
				turnArounds[queue[0]-1] = time - processes[queue[0]-1].ArrivalTime
				completions[queue[0]-1] = time
				waitTimes[queue[0]-1] = time - processes[queue[0]-1].BurstDuration - processes[queue[0]-1].ArrivalTime
			}

			// check for idle time
			var idle bool = true
			if queue[len(processes) - 1] == 0 {
				for j := 0; j < len(processes) && queue[j] != 0; j++ {
					if completions[queue[j] - 1] == -1 {
						idle = false
					}
				}
			} else {
				idle = false
			}

			if idle {
				time++
			
				// check new arrival
				if time <= processes[len(processes) - 1].ArrivalTime {
					var newArrival bool = false
					for j := (highestIndex + 1); j < len(processes); j++ {
						if processes[j].ArrivalTime <= time {
							if highestIndex < j {
								highestIndex = j
								newArrival = true
							}
						}

					}

					// add new arrivals to the queue
					if newArrival {
						var index int = 0
						for j := range processes {
							if queue[j] == 0 {
								index = j
								break
							}
						}
						queue[index] = int64(highestIndex + 1)
					}
				}
			}

			// maintain queue structure
			for j := 0; (j < len(processes) - 1) && (queue[j + 1] != 0); j++ {
				var temp int64 = queue[j]
				queue[j] = queue[j + 1]
				queue[j + 1] = temp
			}
		}
	}

	// calculate averages
	for i := range processes {
		totalTurnaround += turnArounds[i]
	}
	for i := range processes {
		totalWait += waitTimes[i]
	}

	// provide output schedule
	for i := range processes {
		schedule[i] = []string{
			fmt.Sprint(processes[i].ProcessID),
			fmt.Sprint(processes[i].Priority),
			fmt.Sprint(processes[i].BurstDuration),
			fmt.Sprint(processes[i].ArrivalTime),
			fmt.Sprint(waitTimes[i]),
			fmt.Sprint(turnArounds[i]),
			fmt.Sprint(completions[i]),
		}

		gantt = append(gantt, TimeSlice{
			PID:   processes[i].ProcessID,
			Start: completions[i] - waitTimes[i],
			Stop:  completions[i] - waitTimes[i] + processes[i].BurstDuration,
		})
	}

	count := float64(len(processes))
	aveWait := float64(totalWait) / count
	aveTurnaround := float64(totalTurnaround) / count
	aveThroughput := count / float64(time)

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

//endregion

//region Output helpers

func outputTitle(w io.Writer, title string) {
	_, _ = fmt.Fprintln(w, strings.Repeat("-", len(title)*2))
	_, _ = fmt.Fprintln(w, strings.Repeat(" ", len(title)/2), title)
	_, _ = fmt.Fprintln(w, strings.Repeat("-", len(title)*2))
}

func outputGantt(w io.Writer, gantt []TimeSlice) {
	_, _ = fmt.Fprintln(w, "Gantt schedule")
	_, _ = fmt.Fprint(w, "|")
	for i := range gantt {
		pid := fmt.Sprint(gantt[i].PID)
		padding := strings.Repeat(" ", (8-len(pid))/2)
		_, _ = fmt.Fprint(w, padding, pid, padding, "|")
	}
	_, _ = fmt.Fprintln(w)
	for i := range gantt {
		_, _ = fmt.Fprint(w, fmt.Sprint(gantt[i].Start), "\t")
		if len(gantt)-1 == i {
			_, _ = fmt.Fprint(w, fmt.Sprint(gantt[i].Stop))
		}
	}
	_, _ = fmt.Fprintf(w, "\n\n")
}

func outputSchedule(w io.Writer, rows [][]string, wait, turnaround, throughput float64) {
	_, _ = fmt.Fprintln(w, "Schedule table")
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"ID", "Priority", "Burst", "Arrival", "Wait", "Turnaround", "Exit"})
	table.AppendBulk(rows)
	table.SetFooter([]string{"", "", "", "",
		fmt.Sprintf("Average\n%.2f", wait),
		fmt.Sprintf("Average\n%.2f", turnaround),
		fmt.Sprintf("Throughput\n%.2f/t", throughput)})
	table.Render()
}

//endregion

//region Loading processes.

var ErrInvalidArgs = errors.New("invalid args")

func loadProcesses(r io.Reader) ([]Process, error) {
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: reading CSV", err)
	}

	processes := make([]Process, len(rows))
	for i := range rows {
		processes[i].ProcessID = mustStrToInt(rows[i][0])
		processes[i].BurstDuration = mustStrToInt(rows[i][1])
		processes[i].ArrivalTime = mustStrToInt(rows[i][2])
		if len(rows[i]) == 4 {
			processes[i].Priority = mustStrToInt(rows[i][3])
		}
	}

	return processes, nil
}

func mustStrToInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return i
}

//endregion
