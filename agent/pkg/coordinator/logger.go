package coordinator

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sync"
)

type CsvLogger struct {
	filename string
	file     *os.File
	writer   *csv.Writer

	wg       *sync.WaitGroup
	Finished chan FinishedTrigger
}

func NewCsvLogger(filename string) (r *CsvLogger, err error) {
	r = new(CsvLogger)
	r.file, err = os.Create(filename)
	if err != nil {
		return
	}
	r.writer = csv.NewWriter(r.file)
	headers := []string{"t", "queue", "total_agents", "dissemination_time"}
	r.writer.Write(headers)
	r.wg = new(sync.WaitGroup)
	r.Finished = make(chan FinishedTrigger, 1000)
	return
}

func (r *CsvLogger) AwaitCompletion() {
	r.wg.Wait()
}

func (r *CsvLogger) Run() context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	r.wg.Add(1)
	go func() {
		log.Println("Logger goroutine running")
		for {
			select {
			case <-ctx.Done():
				{
					log.Println("Logger draining remaining stats to file")
					for {
						select {
						case <-r.Finished:
							// fmt.Println("Finished drain one")
						default:
							log.Println("Logger complete")
							err := r.file.Close()
							if err != nil {
								fmt.Println("Logger error closing file", err)
							}
							r.wg.Done()
							return
						}
					}
				}
			case <-r.Finished:
				// fmt.Println("Finished one")
			}
		}
	}()
	return cancel
}

// func (r *CsvLogger) Report() error {
// 	var records [][]string
// 	for _, row := range rows {
// 		var record []string
// 		for _, column := range r.headers {
// 			if v, ok := row[column]; ok {
// 				record = append(record, v)
// 			} else {
// 				record = append(record, "")
// 			}
// 		}
// 		records = append(records, record)
// 	}
// 	return r.writer.WriteAll(records)
// }
