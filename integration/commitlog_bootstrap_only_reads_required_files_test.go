// +build integration

// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package integration

import (
	"testing"
	"time"

	"github.com/m3db/m3db/retention"
	"github.com/m3db/m3db/storage/bootstrap"
	"github.com/m3db/m3db/storage/bootstrap/bootstrapper"
	bcl "github.com/m3db/m3db/storage/bootstrap/bootstrapper/commitlog"
	"github.com/m3db/m3db/storage/namespace"

	"github.com/stretchr/testify/require"
)

func TestCommitLogBootstrapOnlyReadsRequiredFiles(t *testing.T) {
	if testing.Short() {
		t.SkipNow() // Just skip if we're doing a short run
	}

	// Test setup
	var (
		ropts     = retention.NewOptions().SetRetentionPeriod(12 * time.Hour)
		blockSize = ropts.BlockSize()
	)
	ns1, err := namespace.NewMetadata(testNamespaces[0], namespace.NewOptions().SetRetentionOptions(ropts))
	require.NoError(t, err)
	opts := newTestOptions(t).
		SetCommitLogRetentionPeriod(ropts.RetentionPeriod()).
		SetCommitLogBlockSize(blockSize).
		SetNamespaces([]namespace.Metadata{ns1})

	setup, err := newTestSetup(t, opts)
	require.NoError(t, err)
	defer setup.close()

	commitLogOpts := setup.storageOpts.CommitLogOptions().
		SetFlushInterval(defaultIntegrationTestFlushInterval)
	setup.storageOpts = setup.storageOpts.SetCommitLogOptions(commitLogOpts)

	noOpAll := bootstrapper.NewNoOpAllBootstrapper()
	bsOpts := newDefaulTestResultOptions(setup.storageOpts)
	bclOpts := bcl.NewOptions().
		SetResultOptions(bsOpts).
		SetCommitLogOptions(commitLogOpts)
	bs, err := bcl.NewCommitLogBootstrapper(bclOpts, noOpAll)
	require.NoError(t, err)
	process := bootstrap.NewProcess(bs, bsOpts)
	setup.storageOpts = setup.storageOpts.SetBootstrapProcess(process)

	log := setup.storageOpts.InstrumentOptions().Logger()
	log.Info("commit log bootstrap test")

	// Write test data
	log.Info("generating data")
	now := setup.getNowFn()
	seriesMaps := generateSeriesMaps(30, now.Add(-2*blockSize), now.Add(-blockSize))
	log.Info("writing data")
	writeCommitLogData(t, setup, seriesMaps, testNamespaces[0])
	log.Info("finished writing data")

	// The datapoints in this generated data are within the retention period and
	// would oridinarly be bootstrapped, however, we intentionally write them to a
	// commitlog file that has a timestamp outside of the retention period. This
	// allows us to verify the commitlog bootstrapping logic will not waste time
	// reading commitlog files that are outside of the retention period.
	log.Info("generating data")
	seriesMapsExpiredCommitlog := generateSeriesMaps(30, now.Add(-2*blockSize), now.Add(-blockSize))
	log.Info("writing data to commitlog file with out of range timestamp")
	writeCommitLogDataSpecifiedTS(
		t,
		setup,
		seriesMapsExpiredCommitlog,
		testNamespaces[0],
		now.Add(-2*ropts.RetentionPeriod()),
	)
	log.Info("finished writing data to commitlog file with out of range timestamp")

	setup.setNowFn(now)
	// Start the server with filesystem bootstrapper
	require.NoError(t, setup.startServer())
	log.Debug("server is now up")

	// Stop the server
	defer func() {
		require.NoError(t, setup.stopServer())
		log.Debug("server is now down")
	}()

	// Verify in-memory data match what we expect - all writes from seriesMaps
	// should be present, and none of the writes from seriesMapsExpiredCommitlog
	// should be present.
	metadatasByShard := testSetupMetadatas(t, setup, testNamespaces[0], now.Add(-2*blockSize), now)
	observedSeriesMaps := testSetupToSeriesMaps(t, setup, ns1, metadatasByShard)
	verifySeriesMapsEqual(t, seriesMaps, observedSeriesMaps)
}
