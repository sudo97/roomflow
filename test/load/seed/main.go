// Command seed fills a RoomFlow instance with tenants, rooms and non-overlapping bookings
// so that load tests run against a realistic data volume.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func post(client *http.Client, url, tenant, body string) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Tenant-ID", tenant)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("%s: status %d", url, resp.StatusCode)
	}
	return nil
}

type config struct {
	base                     string
	tenants, rooms, bookings int
	workers                  int
}

func seedRooms(client *http.Client, c config) error {
	for t := 0; t < c.tenants; t++ {
		for r := 1; r <= c.rooms; r++ {
			body := fmt.Sprintf(`{"id":"r%d","name":"Room %d","capacity":4,"price":150000}`, r, r)
			if err := post(client, c.base+"/api/rooms", fmt.Sprintf("hotel-%d", t), body); err != nil {
				return err
			}
		}
	}
	return nil
}

// bookingFor returns tenant and body of the i-th booking: one-night stays that never overlap.
func bookingFor(i int, c config) (string, string) {
	t, r, slot := i%c.tenants, (i/c.tenants)%c.rooms+1, i/(c.tenants*c.rooms)
	in := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, slot)
	body := fmt.Sprintf(`{"room_id":"r%d","guest_name":"Guest %d","guest_email":"g%d@example.com","guests":2,"check_in":"%s","check_out":"%s"}`,
		r, i, i, in.Format("2006-01-02"), in.AddDate(0, 0, 1).Format("2006-01-02"))
	return fmt.Sprintf("hotel-%d", t), body
}

// worker creates the bookings received from jobs and counts failures.
func worker(client *http.Client, c config, jobs <-chan int, failed *atomic.Int64) {
	for i := range jobs {
		tenant, body := bookingFor(i, c)
		if post(client, c.base+"/api/bookings", tenant, body) != nil {
			failed.Add(1)
		}
	}
}

func seedBookings(client *http.Client, c config) int64 {
	var failed atomic.Int64
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < c.workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(client, c, jobs, &failed)
		}()
	}
	for i := 0; i < c.bookings; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return failed.Load()
}

func main() {
	var c config
	flag.StringVar(&c.base, "url", "http://localhost:8080", "RoomFlow base URL")
	flag.IntVar(&c.tenants, "tenants", 20, "number of hotels")
	flag.IntVar(&c.rooms, "rooms", 10, "rooms per hotel")
	flag.IntVar(&c.bookings, "bookings", 20000, "total bookings")
	flag.IntVar(&c.workers, "workers", 16, "concurrent requests")
	flag.Parse()

	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	if err := seedRooms(client, c); err != nil {
		log.Fatal(err)
	}
	failed := seedBookings(client, c)
	fmt.Printf("seeded %d tenants x %d rooms, %d bookings (%d failed) in %s\n", c.tenants, c.rooms, c.bookings, failed, time.Since(start).Round(time.Millisecond))
}
