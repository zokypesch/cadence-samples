package main

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/zokypesch/cadence-samples/cmd/samples/common"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

func init() {
	activity.Register(SignalWithStartMutexWorkflowActivity)
	workflow.Register(MutexWorkflow)
	workflow.Register(SampleWorkflowWithMutex)
}

const (
	// AcquireLockSignalName signal channel name for lock acquisition
	AcquireLockSignalName = "acquire-lock-event"
	// RequestLockSignalName channel name for request lock
	RequestLockSignalName = "request-lock-event"
)

// UnlockFunc ...
type UnlockFunc func() error

// Mutex - cadence mutex
type Mutex struct {
	currentWorkflowID string
	lockNamespace     string
}

// NewMutex initializes cadence mutex
func NewMutex(currentWorkflowID string, lockNamespace string) *Mutex {
	return &Mutex{
		currentWorkflowID: currentWorkflowID,
		lockNamespace:     lockNamespace,
	}
}

// Lock - locks mutex
func (s *Mutex) Lock(ctx workflow.Context,
	resourceID string, unlockTimeout time.Duration) (UnlockFunc, error) {

	activityCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: time.Minute * 1,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			ExpirationInterval: time.Minute * 10,
			MaximumAttempts:    5,
		},
	})

	var releaseLockChannelName string
	var execution workflow.Execution
	err := workflow.ExecuteLocalActivity(activityCtx,
		SignalWithStartMutexWorkflowActivity, s.lockNamespace,
		resourceID, s.currentWorkflowID, unlockTimeout).Get(ctx, &execution)
	if err != nil {
		return nil, err
	}
	workflow.GetSignalChannel(ctx, AcquireLockSignalName).
		Receive(ctx, &releaseLockChannelName)

	unlockFunc := func() error {
		return workflow.SignalExternalWorkflow(ctx, execution.ID, execution.RunID,
			releaseLockChannelName, "releaseLock").Get(ctx, nil)
	}
	return unlockFunc, nil
}

// MutexWorkflow used for locking a resource
func MutexWorkflow(
	ctx workflow.Context,
	namespace string,
	resourceID string,
	unlockTimeout time.Duration,
) error {
	currentWorkflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	if currentWorkflowID == "default-test-workflow-id" {
		// unit testing hack, see https://github.com/uber-go/cadence-client/issues/663
		workflow.Sleep(ctx, 10*time.Millisecond)
	}
	logger := workflow.GetLogger(ctx).With(zap.String("currentWorkflowID", currentWorkflowID))
	logger.Info("started")
	var ack string
	requestLockCh := workflow.GetSignalChannel(ctx, RequestLockSignalName)
	for {
		var senderWorkflowID string
		if !requestLockCh.ReceiveAsync(&senderWorkflowID) {
			logger.Info("no more signals")
			break
		}
		var releaseLockChannelName string
		_ = workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
			return _generateUnlockChannelName(senderWorkflowID)
		}).Get(&releaseLockChannelName)
		logger := logger.With(zap.String("releaseLockChannelName", releaseLockChannelName))
		logger.Info("generated release lock channel name")
		// Send release lock channel name back to a senderWorkflowID, so that it can
		// release the lock using release lock channel name
		err := workflow.SignalExternalWorkflow(ctx, senderWorkflowID, "",
			AcquireLockSignalName, releaseLockChannelName).Get(ctx, nil)
		if err != nil {
			// .Get(ctx, nil) blocks until the signal is sent.
			// If the senderWorkflowID is closed (terminated/canceled/timeouted/completed/etc), this would return error.
			// In this case we release the lock immediately instead of failing the mutex workflow.
			// Mutex workflow failing would lead to all workflows that have sent requestLock will be waiting.
			logger.With(zap.Error(err)).Info("SignalExternalWorkflow error")
			continue
		}
		logger.With(zap.Error(err)).Info("signaled external workflow")
		selector := workflow.NewSelector(ctx)
		selector.AddFuture(workflow.NewTimer(ctx, unlockTimeout), func(f workflow.Future) {
			logger.Info("unlockTimeout exceeded")
		})
		selector.AddReceive(workflow.GetSignalChannel(ctx, releaseLockChannelName), func(c workflow.Channel, more bool) {
			c.Receive(ctx, &ack)
			logger.Info("release signal received")
		})
		selector.Select(ctx)
	}
	return nil
}

// SignalWithStartMutexWorkflowActivity ...
func SignalWithStartMutexWorkflowActivity(
	ctx context.Context,
	namespace string,
	resourceID string,
	senderWorkflowID string,
	unlockTimeout time.Duration,
) (*workflow.Execution, error) {

	h := ctx.Value(_sampleHelperContextKey).(*common.SampleHelper)
	workflowID := fmt.Sprintf(
		"%s:%s:%s",
		"mutex",
		namespace,
		resourceID,
	)
	workflowOptions := client.StartWorkflowOptions{
		ID:                              workflowID,
		TaskList:                        ApplicationName,
		ExecutionStartToCloseTimeout:    time.Hour,
		DecisionTaskStartToCloseTimeout: time.Hour,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			ExpirationInterval: time.Minute * 10,
			MaximumAttempts:    5,
		},
		WorkflowIDReusePolicy: client.WorkflowIDReusePolicyAllowDuplicate,
	}
	we := h.SignalWithStartWorkflowWithCtx(
		ctx, workflowID, RequestLockSignalName, senderWorkflowID,
		workflowOptions, MutexWorkflow, namespace, resourceID, unlockTimeout)
	return we, nil
}

// _generateUnlockChannelName generates release lock channel name
func _generateUnlockChannelName(senderWorkflowID string) string {
	return fmt.Sprintf("unlock-event-%s", senderWorkflowID)
}

// MockMutexLock stubs cadence mutex.Lock call
func MockMutexLock(env *testsuite.TestWorkflowEnvironment, resourceID string, mockError error) {
	mockExecution := &workflow.Execution{ID: "mockID", RunID: "mockRunID"}
	env.OnActivity(SignalWithStartMutexWorkflowActivity,
		mock.Anything, mock.Anything, resourceID, mock.Anything, mock.Anything).
		Return(mockExecution, mockError)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(AcquireLockSignalName, "mockReleaseLockChannelName")
	}, time.Millisecond*0)
	if mockError == nil {
		env.OnSignalExternalWorkflow(mock.Anything, mock.Anything, mockExecution.RunID,
			mock.Anything, mock.Anything).Return(nil)
	}
}

func SampleWorkflowWithMutex(
	ctx workflow.Context,
	resourceID string,
) error {
	currentWorkflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	logger := workflow.GetLogger(ctx).
		With(zap.String("currentWorkflowID", currentWorkflowID)).
		With(zap.String("resourceID", resourceID))
	logger.Info("started")

	mutex := NewMutex(currentWorkflowID, "TestUseCase")
	unlockFunc, err := mutex.Lock(ctx, resourceID, 10*time.Minute)
	if err != nil {
		return err
	}
	logger.Info("resource locked")

	// emulate long running process
	logger.Info("critical operation started")
	workflow.Sleep(ctx, 10*time.Second)
	logger.Info("critical operation finished")

	unlockFunc()

	logger.Info("finished")
	return nil
}
