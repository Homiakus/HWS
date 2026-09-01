package ipc

import "testing"

func TestHelloInvalidatesCacheAcrossDaemonRestart(t *testing.T) {
	request := HelloRequest{ClientProtocol: ProtocolVersion, ClientInstance: "shell-1"}
	first := HelloResponse{
		ServerProtocol: ProtocolVersion,
		ServerInstance: "daemon-1",
		RevisionEpoch:  "epoch-1",
		Capabilities: map[string]Capability{
			"WindowObservation": {Level: CapabilitySupported},
		},
	}
	if err := ValidateHello(request, first); err != nil {
		t.Fatalf("ValidateHello: %v", err)
	}
	cache := CacheIdentity{ServerInstance: first.ServerInstance, RevisionEpoch: first.RevisionEpoch}
	if !cache.ValidFor(first) {
		t.Fatal("cache should be valid for the same daemon instance")
	}

	restarted := first
	restarted.ServerInstance = "daemon-2"
	if cache.ValidFor(restarted) {
		t.Fatal("cache must be invalid after daemon owner/instance change")
	}
}

func TestMutationRequiresOperationKey(t *testing.T) {
	request := MutationRequest{WorkspaceID: "local-dev", DefinitionRevision: "v1"}
	if err := request.Validate(); err == nil {
		t.Fatal("expected missing operation key error")
	}
	request.OperationKey = "activate:local-dev:v1:1"
	if err := request.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestHelloRejectsProtocolMismatch(t *testing.T) {
	response := HelloResponse{ServerProtocol: ProtocolVersion, ServerInstance: "daemon", RevisionEpoch: "epoch"}
	if err := ValidateHello(HelloRequest{ClientProtocol: ProtocolVersion + 1, ClientInstance: "shell"}, response); err == nil {
		t.Fatal("expected protocol mismatch")
	}
}
