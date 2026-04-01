package handlers

import (
	"cmdb-v2/pkg/models"
	"testing"

	"gorm.io/gorm"
)

func TestSanitizeExportPayloadRemovesBrokenRelations(t *testing.T) {
	clusterID := uint(1)
	hostID := uint(10)
	appID := uint(20)
	missingHostID := uint(999)
	missingAppID := uint(998)
	missingDomainID := uint(997)

	payload := exportPayload{
		Version: "cmdb-export-v1",
		Clusters: []models.Cluster{
			{Model: modelWithID(1), Name: "cluster-a"},
		},
		Hosts: []models.Host{
			{Model: modelWithID(hostID), Name: "host-a", ClusterID: &clusterID},
			{Model: modelWithID(11), Name: "host-b", ClusterID: uintPtr(12345)},
		},
		Apps: []models.App{
			{Model: modelWithID(appID), Name: "app-a", HostID: hostID},
			{Model: modelWithID(21), Name: "app-b", HostID: missingHostID},
		},
		Ports: []models.Port{
			{Model: modelWithID(30), AppID: appID, Port: 80, Protocol: "tcp"},
			{Model: modelWithID(31), AppID: missingAppID, Port: 81, Protocol: "tcp"},
		},
		Domains: []models.Domain{
			{Model: modelWithID(40), Domain: "ok.example.com", AppID: &appID, HostID: &hostID},
			{Model: modelWithID(41), Domain: "broken.example.com", AppID: uintPtr(missingAppID), HostID: uintPtr(missingHostID)},
		},
		Dependencies: []models.Dependency{
			{
				Model:        modelWithID(50),
				SourceAppID:  &appID,
				TargetHostID: uintPtr(missingHostID),
				DomainID:     uintPtr(40),
				SourceNode:   "app-a",
				TargetNode:   "host-a",
			},
			{
				Model:       modelWithID(51),
				SourceAppID: uintPtr(missingAppID),
				DomainID:    &missingDomainID,
				SourceNode:  "broken",
				TargetNode:  "broken",
			},
		},
	}

	sanitized := sanitizeExportPayload(payload)

	if len(sanitized.Apps) != 1 {
		t.Fatalf("expected 1 app after sanitize, got %d", len(sanitized.Apps))
	}
	if sanitized.Apps[0].ID != appID {
		t.Fatalf("expected app %d to remain, got %d", appID, sanitized.Apps[0].ID)
	}
	if len(sanitized.Ports) != 1 {
		t.Fatalf("expected 1 port after sanitize, got %d", len(sanitized.Ports))
	}
	if sanitized.Hosts[1].ClusterID != nil {
		t.Fatalf("expected broken host cluster reference to be cleared")
	}
	if sanitized.Domains[1].AppID != nil || sanitized.Domains[1].HostID != nil {
		t.Fatalf("expected broken domain references to be cleared")
	}
	if sanitized.Dependencies[0].TargetHostID != nil {
		t.Fatalf("expected broken dependency host reference to be cleared")
	}
	if sanitized.Dependencies[1].SourceAppID != nil {
		t.Fatalf("expected broken dependency app reference to be cleared")
	}
	if sanitized.Dependencies[1].DomainID != nil {
		t.Fatalf("expected broken dependency domain reference to be cleared")
	}
}

func modelWithID(id uint) gorm.Model {
	return gorm.Model{ID: id}
}
