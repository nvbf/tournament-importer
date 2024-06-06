package sync

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/gin-gonic/gin"
	profixio "github.com/nvbf/tournament-sync/repos/profixio"
)

type SyncService struct {
	firestoreClient *firestore.Client
	firebaseApp     *firebase.App
	profixioService *profixio.Service
}

func NewSyncService(firestoreClient *firestore.Client, firebaseApp *firebase.App, profixioService *profixio.Service) *SyncService {
	return &SyncService{
		firestoreClient: firestoreClient,
		firebaseApp:     firebaseApp,
		profixioService: profixioService,
	}
}

func (s *SyncService) FetchTournaments(c *gin.Context) error {
	ctx := context.Background()
	go s.profixioService.FetchTournaments(ctx, 1)

	c.JSON(http.StatusOK, gin.H{
		"message": "Async function started",
	})
	return nil
}

func (s *SyncService) SyncTournamentMatches(c *gin.Context, slug string) error {
	layout := "2006-01-02 15:04:05"

	t := time.Now()
	t_m := time.Now().Add(-10 * time.Minute)
	now := t.Format(layout)
	now_m := t_m.Format(layout)

	ctx := context.Background()
	lastSync := s.profixioService.GetLastSynced(ctx, slug)
	lastReq := s.profixioService.GetLastRequest(ctx, slug)
	if lastReq == "" {
		lastReq = layout
	}
	lastRequestTime, err := time.Parse(layout, lastReq)
	if err != nil {
		fmt.Println(err)
	}
	newTime := t.Add(0 * time.Hour)
	diff := newTime.Sub(lastRequestTime)
	if diff < 0*time.Second {
		newTime = t.Add(2 * time.Hour)
		diff = newTime.Sub(lastRequestTime)
	}

	log.Printf("Since last req: %s\n", diff)

	if diff < 30*time.Second {
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Seconds since last req: %s", diff),
		})
	} else {
		s.profixioService.SetLastRequest(ctx, slug, now)
		go s.profixioService.FetchMatches(ctx, 1, slug, lastSync, now_m)

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Async function started sync from lastSync: %s", lastSync),
		})
	}
	return nil
}

func (s *SyncService) UpdateCustomTournament(c *gin.Context, slug string, tournament profixio.CustomTournament) error {
	ctx := context.Background()
	go s.profixioService.ProcessCustomTournament(ctx, slug, tournament)

	return nil
}
