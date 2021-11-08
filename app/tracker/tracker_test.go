package tracker

import (
	"context"
	"github.com/cappuccinotm/dastracker/app/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMultiTracker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	trk1 := &trackerMock{chn: make(chan store.Update), name: "trk1"}
	trk2 := &trackerMock{chn: make(chan store.Update), name: "trk2"}

	mtrk, err := NewMultiTracker(ctx, []Interface{trk1, trk2})
	require.NoError(t, err)

	updates := mtrk.Updates()

	trk1.Publish(store.Update{ReceivedFrom: store.Locator{Tracker: "trk1", TaskID: "1"}})
	waitUntilForwarded(trk1.chn, mtrk.chn, 1)
	trk2.Publish(store.Update{ReceivedFrom: store.Locator{Tracker: "trk2", TaskID: "1"}})
	waitUntilForwarded(trk2.chn, mtrk.chn, 2)
	trk1.Publish(store.Update{ReceivedFrom: store.Locator{Tracker: "trk1", TaskID: "2"}})
	waitUntilForwarded(trk1.chn, mtrk.chn, 3)

	assert.Equal(t, store.Update{ReceivedFrom: store.Locator{Tracker: "trk1", TaskID: "1"}}, <-updates)
	assert.Equal(t, store.Update{ReceivedFrom: store.Locator{Tracker: "trk2", TaskID: "1"}}, <-updates)
	assert.Equal(t, store.Update{ReceivedFrom: store.Locator{Tracker: "trk1", TaskID: "2"}}, <-updates)

	cancel()
	err = mtrk.Close(context.Background())
	assert.NoError(t, err)

	assert.Len(t, trk1.CloseCalls(), 1)
	assert.Len(t, trk2.CloseCalls(), 1)
}

func waitUntilForwarded(from, to chan store.Update, numOfQueued int) {
	for {
		if len(from) == 0 && len(to) == numOfQueued {
			return
		}
	}
}

type trackerMock struct {
	InterfaceMock
	chn  chan store.Update
	name string
}

func (t *trackerMock) Name() string {
	return t.name
}

func (t *trackerMock) Updates() <-chan store.Update {
	return t.chn
}

func (t *trackerMock) Publish(upd store.Update) {
	t.chn <- upd
}

func (t *trackerMock) Close(ctx context.Context) error {
	close(t.chn)
	if t.InterfaceMock.CloseFunc == nil {
		t.InterfaceMock.CloseFunc = func(_ context.Context) error { return nil }
	}
	return t.InterfaceMock.Close(ctx)
}
