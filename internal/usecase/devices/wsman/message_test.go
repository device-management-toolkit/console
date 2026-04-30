package wsman

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gwmconfig "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// passthroughCryptor is a minimal test-local security.Cryptor implementation.
// Inlined to avoid the import cycle with internal/mocks.
type passthroughCryptor struct{}

func (passthroughCryptor) Decrypt(cipherText string) (string, error)          { return cipherText, nil }
func (passthroughCryptor) Encrypt(plainText string) (string, error)           { return plainText, nil }
func (passthroughCryptor) EncryptWithKey(plainText, _ string) (string, error) { return plainText, nil }
func (passthroughCryptor) GenerateKey() string                                { return "" }
func (passthroughCryptor) ReadAndDecryptFile(string) (gwmconfig.Configuration, error) {
	return gwmconfig.Configuration{}, nil
}

// TestSetupWsmanClientCancelledContextDoesNotDeadlockWorker is a regression test
// for a deadlock where a caller whose context was canceled would abandon an
// unbuffered resultChan, leaving the single Worker goroutine permanently stuck
// on its send. All subsequent wsman requests would then pile up in requestQueue
// with no one to process them, causing every future wsman-dependent API call
// to time out with 408.
func TestSetupWsmanClientCancelledContextDoesNotDeadlockWorker(t *testing.T) { //nolint:paralleltest // mutates package-level state (requestQueue, queueTickTime, connections)
	// Mutates package-level state (requestQueue, queueTickTime, connections).
	origTick := queueTickTime
	queueTickTime = 1 * time.Millisecond

	t.Cleanup(func() { queueTickTime = origTick })

	guid := "cancel-deadlock-regression-guid"

	t.Cleanup(func() { RemoveConnection(guid) })

	// Pre-populate a CIRA connection entry so the SetupWsmanClient closure
	// deterministically reaches the `resultChan <- connection` send path
	// without any network I/O.
	SetConnectionEntry(guid, &ConnectionEntry{
		IsCIRA: true,
		Timer:  time.AfterFunc(time.Hour, func() {}),
	})

	g := NewGoWSMANMessages(logger.New("error"), passthroughCryptor{})

	device := entity.Device{
		GUID:        guid,
		MPSUsername: "mpsuser",
		Username:    "user",
		Password:    "pw",
	}

	// Call 1: context is pre-canceled and NO worker is running yet, so the
	// closure sits in requestQueue while the caller returns immediately via
	// ctx.Done. The result receiver is abandoned before the closure executes.
	ctxCancelled, cancelFirst := context.WithCancel(context.Background())
	cancelFirst()

	_, err := g.SetupWsmanClient(ctxCancelled, device, false, false)
	require.Error(t, err, "first call must return the cancellation error")

	// Start the Worker AFTER the first caller has abandoned its resultChan.
	// Without the fix, the worker will now dequeue the closure and block
	// forever trying to send on the unbuffered, receiver-less resultChan.
	stopWorker := make(chan struct{})
	workerDone := make(chan struct{})

	go func() {
		defer close(workerDone)

		for {
			select {
			case <-stopWorker:
				return
			case req := <-requestQueue:
				req()
				time.Sleep(queueTickTime)
			}
		}
	}()

	t.Cleanup(func() {
		close(stopWorker)
		// If the worker is deadlocked the test has already failed; don't
		// wait forever for it to exit.
		select {
		case <-workerDone:
		case <-time.After(time.Second):
		}
	})

	// Call 2: a fresh, uncancelled request. If the worker is wedged on the
	// abandoned send from call 1, this call never returns.
	ctx, cancelSecond := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelSecond()

	done := make(chan error, 1)

	go func() {
		_, err := g.SetupWsmanClient(ctx, device, false, false)
		done <- err
	}()

	select {
	case err := <-done:
		assert.NoError(t, err, "second call must succeed after a canceled first call")
	case <-time.After(3 * time.Second):
		t.Fatal("Worker is deadlocked: SetupWsmanClient did not process a new request after a canceled one")
	}
}

// TestDestroyWsmanClient_PreservesCIRAEntry is a regression test for a bug
// where a PATCH on a CIRA-managed device would call DestroyWsmanClient and
// remove the tunnel registration, orphaning the live socket: the underlying
// TCP connection stayed open (so AMT never reconnected) but every subsequent
// SetupWsmanClient returned ErrCIRADeviceNotConnected. The CIRA tunnel is
// re-registered only on a fresh APF auth, which the orphan state prevents.
func TestDestroyWsmanClient_PreservesCIRAEntry(t *testing.T) {
	t.Parallel()

	guid := "destroy-preserves-cira-entry"

	t.Cleanup(func() { RemoveConnection(guid) })

	SetConnectionEntry(guid, &ConnectionEntry{
		IsCIRA: true,
		Timer:  time.AfterFunc(time.Hour, func() {}),
	})

	g := NewGoWSMANMessages(logger.New("error"), passthroughCryptor{})
	g.DestroyWsmanClient(dto.Device{GUID: guid})

	require.NotNil(t, GetConnectionEntry(guid),
		"DestroyWsmanClient must preserve CIRA entries; the TCP tunnel is the only registration path")
}

func TestDestroyWsmanClient_RemovesNonCIRAEntry(t *testing.T) {
	t.Parallel()

	guid := "destroy-removes-non-cira-entry"

	t.Cleanup(func() { RemoveConnection(guid) })

	SetConnectionEntry(guid, &ConnectionEntry{
		IsCIRA: false,
		Timer:  time.AfterFunc(time.Hour, func() {}),
	})

	g := NewGoWSMANMessages(logger.New("error"), passthroughCryptor{})
	g.DestroyWsmanClient(dto.Device{GUID: guid})

	require.Nil(t, GetConnectionEntry(guid),
		"DestroyWsmanClient must remove non-CIRA entries so the cache is rebuilt with current credentials")
}

func TestDestroyWsmanClient_MissingEntryIsNoop(t *testing.T) {
	t.Parallel()

	g := NewGoWSMANMessages(logger.New("error"), passthroughCryptor{})
	// Should not panic when the entry is absent.
	g.DestroyWsmanClient(dto.Device{GUID: "destroy-missing-entry"})
}
