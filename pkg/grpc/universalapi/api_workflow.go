/*
Copyright 2023 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package universalapi

import (
	"context"
	"errors"
	"unicode"

	"github.com/microsoft/durabletask-go/api"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/dapr/components-contrib/workflows"
	"github.com/dapr/dapr/pkg/messages"
	runtimev1pb "github.com/dapr/dapr/pkg/proto/runtime/v1"
)

// GetWorkflowBeta1 is the API handler for getting workflow details
func (a *UniversalAPI) GetWorkflowBeta1(ctx context.Context, in *runtimev1pb.GetWorkflowRequest) (*runtimev1pb.GetWorkflowResponse, error) {
	if err := a.validateInstanceID(in.InstanceId, false /* isCreate */); err != nil {
		a.Logger.Debug(err)
		return &runtimev1pb.GetWorkflowResponse{}, err
	}

	// Workflow requires actors to be ready
	a.WaitForActorsReady(ctx)

	workflowComponent, err := a.getWorkflowComponent(in.WorkflowComponent)
	if err != nil {
		a.Logger.Debug(err)
		return &runtimev1pb.GetWorkflowResponse{}, err
	}

	req := workflows.GetRequest{
		InstanceID: in.InstanceId,
	}
	response, err := workflowComponent.Get(ctx, &req)
	if err != nil {
		if errors.Is(err, api.ErrInstanceNotFound) {
			err = messages.ErrWorkflowInstanceNotFound.WithFormat(in.InstanceId, err)
		} else {
			err = messages.ErrWorkflowGetResponse.WithFormat(in.InstanceId, err)
		}
		a.Logger.Debug(err)
		return &runtimev1pb.GetWorkflowResponse{}, err
	}

	res := &runtimev1pb.GetWorkflowResponse{
		InstanceId:    response.Workflow.InstanceID,
		WorkflowName:  response.Workflow.WorkflowName,
		CreatedAt:     timestamppb.New(response.Workflow.CreatedAt),
		LastUpdatedAt: timestamppb.New(response.Workflow.LastUpdatedAt),
		RuntimeStatus: response.Workflow.RuntimeStatus,
		Properties:    response.Workflow.Properties,
	}
	return res, nil
}

// StartWorkflowBeta1 is the API handler for starting a workflow
func (a *UniversalAPI) StartWorkflowBeta1(ctx context.Context, in *runtimev1pb.StartWorkflowRequest) (*runtimev1pb.StartWorkflowResponse, error) {
	if err := a.validateInstanceID(in.InstanceId, true /* isCreate */); err != nil {
		a.Logger.Debug(err)
		return &runtimev1pb.StartWorkflowResponse{}, err
	}

	if in.WorkflowName == "" {
		err := messages.ErrWorkflowNameMissing
		a.Logger.Debug(err)
		return &runtimev1pb.StartWorkflowResponse{}, err
	}

	// Workflow requires actors to be ready
	a.WaitForActorsReady(ctx)

	workflowComponent, err := a.getWorkflowComponent(in.WorkflowComponent)
	if err != nil {
		a.Logger.Debug(err)
		return &runtimev1pb.StartWorkflowResponse{}, err
	}

	req := workflows.StartRequest{
		InstanceID:    in.InstanceId,
		Options:       in.Options,
		WorkflowName:  in.WorkflowName,
		WorkflowInput: in.Input,
	}

	resp, err := workflowComponent.Start(ctx, &req)
	if err != nil {
		err := messages.ErrStartWorkflow.WithFormat(in.WorkflowName, err)
		a.Logger.Debug(err)
		return &runtimev1pb.StartWorkflowResponse{}, err
	}
	ret := &runtimev1pb.StartWorkflowResponse{
		InstanceId: resp.InstanceID,
	}
	return ret, nil
}

// TerminateWorkflowBeta1 is the API handler for terminating a workflow
func (a *UniversalAPI) TerminateWorkflowBeta1(ctx context.Context, in *runtimev1pb.TerminateWorkflowRequest) (*emptypb.Empty, error) {
	emptyResponse := &emptypb.Empty{}
	if err := a.validateInstanceID(in.InstanceId, false /* isCreate */); err != nil {
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	// Workflow requires actors to be ready
	a.WaitForActorsReady(ctx)

	workflowComponent, err := a.getWorkflowComponent(in.WorkflowComponent)
	if err != nil {
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	req := &workflows.TerminateRequest{
		InstanceID: in.InstanceId,
	}
	if err := workflowComponent.Terminate(ctx, req); err != nil {
		if errors.Is(err, api.ErrInstanceNotFound) {
			err = messages.ErrWorkflowInstanceNotFound.WithFormat(in.InstanceId, err)
		} else {
			err = messages.ErrTerminateWorkflow.WithFormat(in.InstanceId, err)
		}
		a.Logger.Debug(err)
		return emptyResponse, err
	}
	return emptyResponse, nil
}

// RaiseEventWorkflowBeta1 is the API handler for raising an event to a workflow
func (a *UniversalAPI) RaiseEventWorkflowBeta1(ctx context.Context, in *runtimev1pb.RaiseEventWorkflowRequest) (*emptypb.Empty, error) {
	emptyResponse := &emptypb.Empty{}
	if err := a.validateInstanceID(in.InstanceId, false /* isCreate */); err != nil {
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	if in.EventName == "" {
		err := messages.ErrMissingWorkflowEventName
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	// Workflow requires actors to be ready
	a.WaitForActorsReady(ctx)

	workflowComponent, err := a.getWorkflowComponent(in.WorkflowComponent)
	if err != nil {
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	req := workflows.RaiseEventRequest{
		InstanceID: in.InstanceId,
		EventName:  in.EventName,
		EventData:  in.EventData,
	}

	err = workflowComponent.RaiseEvent(ctx, &req)
	if err != nil {
		err = messages.ErrRaiseEventWorkflow.WithFormat(in.InstanceId, err)
		a.Logger.Debug(err)
		return emptyResponse, err
	}
	return emptyResponse, nil
}

// PauseWorkflowBeta1 is the API handler for pausing a workflow
func (a *UniversalAPI) PauseWorkflowBeta1(ctx context.Context, in *runtimev1pb.PauseWorkflowRequest) (*emptypb.Empty, error) {
	emptyResponse := &emptypb.Empty{}
	if err := a.validateInstanceID(in.InstanceId, false /* isCreate */); err != nil {
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	// Workflow requires actors to be ready
	a.WaitForActorsReady(ctx)

	workflowComponent, err := a.getWorkflowComponent(in.WorkflowComponent)
	if err != nil {
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	req := &workflows.PauseRequest{
		InstanceID: in.InstanceId,
	}
	if err := workflowComponent.Pause(ctx, req); err != nil {
		err = messages.ErrPauseWorkflow.WithFormat(in.InstanceId, err)
		a.Logger.Debug(err)
		return emptyResponse, err
	}
	return emptyResponse, nil
}

// ResumeWorkflowBeta1 is the API handler for resuming a workflow
func (a *UniversalAPI) ResumeWorkflowBeta1(ctx context.Context, in *runtimev1pb.ResumeWorkflowRequest) (*emptypb.Empty, error) {
	emptyResponse := &emptypb.Empty{}
	if err := a.validateInstanceID(in.InstanceId, false /* isCreate */); err != nil {
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	// Workflow requires actors to be ready
	a.WaitForActorsReady(ctx)

	workflowComponent, err := a.getWorkflowComponent(in.WorkflowComponent)
	if err != nil {
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	req := &workflows.ResumeRequest{
		InstanceID: in.InstanceId,
	}
	if err := workflowComponent.Resume(ctx, req); err != nil {
		err = messages.ErrResumeWorkflow.WithFormat(in.InstanceId, err)
		a.Logger.Debug(err)
		return emptyResponse, err
	}
	return emptyResponse, nil
}

// PurgeWorkflowBeta1 is the API handler for purging a workflow
func (a *UniversalAPI) PurgeWorkflowBeta1(ctx context.Context, in *runtimev1pb.PurgeWorkflowRequest) (*emptypb.Empty, error) {
	emptyResponse := &emptypb.Empty{}
	if err := a.validateInstanceID(in.InstanceId, false /* isCreate */); err != nil {
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	// Workflow requires actors to be ready
	a.WaitForActorsReady(ctx)

	workflowComponent, err := a.getWorkflowComponent(in.WorkflowComponent)
	if err != nil {
		a.Logger.Debug(err)
		return emptyResponse, err
	}

	req := workflows.PurgeRequest{
		InstanceID: in.InstanceId,
	}

	err = workflowComponent.Purge(ctx, &req)
	if err != nil {
		if errors.Is(err, api.ErrInstanceNotFound) {
			err = messages.ErrWorkflowInstanceNotFound.WithFormat(in.InstanceId, err)
		} else {
			err = messages.ErrPurgeWorkflow.WithFormat(in.InstanceId, err)
		}
		a.Logger.Debug(err)
		return emptyResponse, err
	}
	return emptyResponse, nil
}

// GetWorkflowAlpha1 is the API handler for getting workflow details
func (a *UniversalAPI) GetWorkflowAlpha1(ctx context.Context, in *runtimev1pb.GetWorkflowRequest) (*runtimev1pb.GetWorkflowResponse, error) {
	return a.GetWorkflowBeta1(ctx, in)
}

// StartWorkflowAlpha1 is the API handler for starting a workflow
func (a *UniversalAPI) StartWorkflowAlpha1(ctx context.Context, in *runtimev1pb.StartWorkflowRequest) (*runtimev1pb.StartWorkflowResponse, error) {
	return a.StartWorkflowBeta1(ctx, in)
}

// TerminateWorkflowAlpha1 is the API handler for terminating a workflow
func (a *UniversalAPI) TerminateWorkflowAlpha1(ctx context.Context, in *runtimev1pb.TerminateWorkflowRequest) (*emptypb.Empty, error) {
	return a.TerminateWorkflowBeta1(ctx, in)
}

// RaiseEventWorkflowAlpha1 is the API handler for raising an event to a workflow
func (a *UniversalAPI) RaiseEventWorkflowAlpha1(ctx context.Context, in *runtimev1pb.RaiseEventWorkflowRequest) (*emptypb.Empty, error) {
	return a.RaiseEventWorkflowBeta1(ctx, in)
}

// PauseWorkflowAlpha1 is the API handler for pausing a workflow
func (a *UniversalAPI) PauseWorkflowAlpha1(ctx context.Context, in *runtimev1pb.PauseWorkflowRequest) (*emptypb.Empty, error) {
	return a.PauseWorkflowBeta1(ctx, in)
}

// ResumeWorkflowAlpha1 is the API handler for resuming a workflow
func (a *UniversalAPI) ResumeWorkflowAlpha1(ctx context.Context, in *runtimev1pb.ResumeWorkflowRequest) (*emptypb.Empty, error) {
	return a.ResumeWorkflowBeta1(ctx, in)
}

// PurgeWorkflowAlpha1 is the API handler for purging a workflow
func (a *UniversalAPI) PurgeWorkflowAlpha1(ctx context.Context, in *runtimev1pb.PurgeWorkflowRequest) (*emptypb.Empty, error) {
	return a.PurgeWorkflowBeta1(ctx, in)
}

func (a *UniversalAPI) validateInstanceID(instanceID string, isCreate bool) error {
	if instanceID == "" {
		return messages.ErrMissingOrEmptyInstance
	}

	if isCreate {
		// Limit the length of the instance ID to avoid potential conflicts with state stores that have restrictive key limits.
		const maxInstanceIDLength = 64
		if len(instanceID) > maxInstanceIDLength {
			return messages.ErrInstanceIDTooLong.WithFormat(maxInstanceIDLength)
		}

		// Check to see if the instance ID contains invalid characters. Valid characters are letters, digits, dashes, and underscores.
		// See https://github.com/dapr/dapr/issues/6156 for more context on why we check this.
		for _, c := range instanceID {
			if !unicode.IsLetter(c) && c != '_' && c != '-' && !unicode.IsDigit(c) {
				return messages.ErrInvalidInstanceID.WithFormat(instanceID)
			}
		}
	}
	return nil
}

func (a *UniversalAPI) getWorkflowComponent(componentName string) (workflows.Workflow, error) {
	if componentName == "" {
		return nil, messages.ErrNoOrMissingWorkflowComponent
	}

	workflowComponent, ok := a.CompStore.GetWorkflow(componentName)
	if !ok {
		err := messages.ErrWorkflowComponentDoesNotExist.WithFormat(componentName)
		a.Logger.Debug(err)
		return nil, err
	}
	return workflowComponent, nil
}
