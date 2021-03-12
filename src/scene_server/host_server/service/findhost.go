/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"encoding/json"
	"net/http"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	meta "configcenter/src/common/metadata"
	"configcenter/src/common/util"

	"github.com/emicklei/go-restful"
)

func (s *Service) FindModuleHost(req *restful.Request, resp *restful.Response) {
	srvData := s.newSrvComm(req.Request.Header)
	defErr := srvData.ccErr

	body := new(meta.HostModuleFind)
	if err := json.NewDecoder(req.Request.Body).Decode(body); err != nil {
		blog.Errorf("find host failed with decode body err: %#v, rid:%s", err, srvData.rid)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommJSONUnmarshalFailed)})
		return
	}

	host, err := srvData.lgc.FindHostByModuleIDs(srvData.ctx, body, false)
	if err != nil {
		blog.Errorf("find host failed, err: %#v, input:%#v, rid:%s", err, body, srvData.rid)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrHostGetFail)})
		return
	}

	_ = resp.WriteEntity(meta.SearchHostResult{
		BaseResp: meta.SuccessBaseResp,
		Data:     *host,
	})
}

// ListHosts list host under business specified by path parameter
func (s *Service) ListBizHosts(req *restful.Request, resp *restful.Response) {
	header := req.Request.Header
	ctx := util.NewContextFromHTTPHeader(header)
	rid := util.ExtractRequestIDFromContext(ctx)
	srvData := s.newSrvComm(header)
	defErr := srvData.ccErr

	parameter := meta.ListHostsParameter{}
	if err := json.NewDecoder(req.Request.Body).Decode(&parameter); err != nil {
		blog.Errorf("ListBizHosts failed, decode body failed, err: %#v, rid:%s", err, rid)
		ccErr := defErr.Error(common.CCErrCommJSONUnmarshalFailed)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: ccErr})
		return
	}
	if key, err := parameter.Validate(); err != nil {
		blog.ErrorJSON("ListBizHosts failed, Validate failed,parameter:%s, err: %s, rid:%s", parameter, err, srvData.rid)
		ccErr := srvData.ccErr.CCErrorf(common.CCErrCommParamsInvalid, key)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: ccErr})
		return
	}
	bizID, err := util.GetInt64ByInterface(req.PathParameter("appid"))
	if err != nil {
		ccErr := defErr.Errorf(common.CCErrCommParamsInvalid, common.BKAppIDField)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: ccErr})
		return
	}
	if bizID == 0 {
		ccErr := defErr.Errorf(common.CCErrCommParamsInvalid, common.BKAppIDField)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: ccErr})
		return
	}
	header.Add(common.ReadPreferencePolicyKey, common.SecondaryPreference)
	hostResult, ccErr := s.listBizHosts(req.Request.Header, bizID, parameter)
	if ccErr != nil {
		blog.ErrorJSON("ListBizHosts failed, listBizHosts failed, bizID: %s, parameter: %s, err: %s, rid:%s", bizID, parameter, ccErr.Error(), rid)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: ccErr})
		return
	}
	result := meta.NewSuccessResponse(hostResult)
	_ = resp.WriteAsJson(result)
}

func (s *Service) listBizHosts(header http.Header, bizID int64, parameter meta.ListHostsParameter) (result *meta.ListHostResult, ccErr errors.CCErrorCoder) {
	ctx := util.NewContextFromHTTPHeader(header)
	rid := util.ExtractRequestIDFromContext(ctx)
	srvData := s.newSrvComm(header)
	defErr := srvData.ccErr

	if parameter.Page.IsIllegal() {
		blog.Errorf("ListBizHosts failed, page limit %d illegal, rid:%s", parameter.Page.Limit, srvData.rid)
		return result, defErr.CCErrorf(common.CCErrCommParamsInvalid, "page.limit")
	}

	if parameter.SetIDs != nil && len(parameter.SetIDs) != 0 && parameter.SetCond != nil && len(parameter.SetCond) != 0 {
		blog.Errorf("ListBizHosts failed, bk_set_ids and set_cond can't both be set, rid:%s", srvData.rid)
		return result, defErr.CCErrorf(common.CCErrCommParamsInvalid, "bk_set_ids and set_cond can't both be set")
	}

	var setIDs []int64
	var err error
	if parameter.SetIDs != nil {
		setIDs = parameter.SetIDs
	} else {
		setIDs, err = srvData.lgc.GetSetIDByCond(srvData.ctx, parameter.SetCond)
		if err != nil {
			blog.ErrorJSON("ListBizHosts failed, GetSetIDByCond %s failed, error: %s, rid:%s", parameter.SetCond, err.Error(), srvData.rid)
			return result, defErr.CCError(common.CCErrCommHTTPDoRequestFailed)
		}
		if len(setIDs) == 0 {
			return &meta.ListHostResult{
				Count: 0,
				Info:  []map[string]interface{}{},
			}, nil
		}
	}

	option := &meta.ListHosts{
		BizID:              bizID,
		SetIDs:             setIDs,
		ModuleIDs:          parameter.ModuleIDs,
		HostPropertyFilter: parameter.HostPropertyFilter,
		Fields:             parameter.Fields,
		Page:               parameter.Page,
	}
	hostResult, err := s.CoreAPI.CoreService().Host().ListHosts(ctx, header, option)
	if err != nil {
		blog.Errorf("find host failed, err: %s, input:%#v, rid:%s", err.Error(), parameter, rid)
		return result, defErr.CCError(common.CCErrHostGetFail)
	}
	return hostResult, nil
}

// ListHostsWithNoBiz list host for no biz case merely
func (s *Service) ListHostsWithNoBiz(req *restful.Request, resp *restful.Response) {
	header := req.Request.Header
	ctx := util.NewContextFromHTTPHeader(header)
	rid := util.ExtractRequestIDFromContext(ctx)
	srvData := s.newSrvComm(header)
	defErr := srvData.ccErr

	parameter := &meta.ListHostsWithNoBizParameter{}
	if err := json.NewDecoder(req.Request.Body).Decode(parameter); err != nil {
		blog.Errorf("ListHostsWithNoBiz failed, decode body failed, err: %#v, rid:%s", err, srvData.rid)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommJSONUnmarshalFailed)})
		return
	}

	if parameter.Page.IsIllegal() {
		blog.Errorf("ListHostsWithNoBiz failed, page limit %d illegal, rid:%s", parameter.Page.Limit, srvData.rid)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.CCErrorf(common.CCErrCommParamsInvalid, "page.limit")})
		return
	}
	option := &meta.ListHosts{
		HostPropertyFilter: parameter.HostPropertyFilter,
		Fields:             parameter.Fields,
		Page:               parameter.Page,
	}
	host, err := s.CoreAPI.CoreService().Host().ListHosts(ctx, header, option)
	if err != nil {
		blog.Errorf("find host failed, err: %s, input:%#v, rid:%s", err.Error(), parameter, rid)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrHostGetFail)})
		return
	}

	result := meta.NewSuccessResponse(host)
	_ = resp.WriteEntity(result)
}

// ListBizHostsTopo list hosts under business specified by path parameter with their topology information
func (s *Service) ListBizHostsTopo(req *restful.Request, resp *restful.Response) {
	header := req.Request.Header
	ctx := util.NewContextFromHTTPHeader(header)
	rid := util.ExtractRequestIDFromContext(ctx)
	srvData := s.newSrvComm(header)
	defErr := srvData.ccErr

	parameter := &meta.ListHostsWithNoBizParameter{}
	if err := json.NewDecoder(req.Request.Body).Decode(parameter); err != nil {
		blog.Errorf("ListHostByTopoNode failed, decode body failed, err: %#v, rid:%s", err, srvData.rid)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommJSONUnmarshalFailed)})
		return
	}
	if key, err := parameter.Validate(); err != nil {
		blog.ErrorJSON("ListHostByTopoNode failed, Validate failed,parameter:%s, err: %s, rid:%s", parameter, err, srvData.rid)
		ccErr := srvData.ccErr.CCErrorf(common.CCErrCommParamsInvalid, key)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: ccErr})
		return
	}
	bizID, err := util.GetInt64ByInterface(req.PathParameter("bk_biz_id"))
	if err != nil {
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Errorf(common.CCErrCommParamsInvalid, "bk_app_id")})
		return
	}
	if bizID == 0 {
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Errorf(common.CCErrCommParamsInvalid, "bk_app_id")})
		return
	}

	if parameter.Page.IsIllegal() {
		blog.Errorf("ListHostByTopoNode failed, page limit %d illegal, rid:%s", parameter.Page.Limit, srvData.rid)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.CCErrorf(common.CCErrCommParamsInvalid, "page.limit")})
		return
	}

	// search all hosts
	header.Add(common.ReadPreferencePolicyKey, common.SecondaryPreference)
	option := &meta.ListHosts{
		BizID:              bizID,
		HostPropertyFilter: parameter.HostPropertyFilter,
		Fields:             parameter.Fields,
		Page:               parameter.Page,
	}
	hosts, err := s.CoreAPI.CoreService().Host().ListHosts(ctx, header, option)
	if err != nil {
		blog.Errorf("find host failed, err: %s, input:%#v, rid:%s", err.Error(), parameter, rid)
		_ = resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrHostGetFail)})
		return
	}

	if len(hosts.Info) == 0 {
		result := meta.NewSuccessResponse(hosts)
		_ = resp.WriteAsJson(result)
		return
	}

	// search all hosts' host module relations
	hostIDs := make([]int64, 0)
	for _, host := range hosts.Info {
		hostID, err := util.GetInt64ByInterface(host[common.BKHostIDField])
		if err != nil {
			blog.ErrorJSON("host: %s bk_host_id field invalid, rid:%s", host, rid)
			_ = resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: err})
			return
		}
		hostIDs = append(hostIDs, hostID)
	}
	relationCond := meta.HostModuleRelationRequest{
		ApplicationID: bizID,
		HostIDArr:     hostIDs,
		Fields:        []string{common.BKSetIDField, common.BKModuleIDField, common.BKHostIDField},
	}
	relations, err := srvData.lgc.GetConfigByCond(srvData.ctx, relationCond)
	if nil != err {
		blog.ErrorJSON("read host module relation error: %s, input: %s, rid: %s", err, hosts, srvData.rid)
		_ = resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: err})
		return
	}

	// search all module and set info
	setIDs := make([]int64, 0)
	moduleIDs := make([]int64, 0)
	relation := make(map[int64]map[int64][]int64)
	for _, r := range relations {
		setIDs = append(setIDs, r.SetID)
		moduleIDs = append(moduleIDs, r.ModuleID)
		if setModule, ok := relation[r.HostID]; ok {
			setModule[r.SetID] = append(setModule[r.SetID], r.ModuleID)
			relation[r.HostID] = setModule
		} else {
			setModule := make(map[int64][]int64)
			setModule[r.SetID] = append(setModule[r.SetID], r.ModuleID)
			relation[r.HostID] = setModule
		}
	}
	setIDs = util.IntArrayUnique(setIDs)
	moduleIDs = util.IntArrayUnique(moduleIDs)

	cond := map[string]interface{}{common.BKSetIDField: map[string]interface{}{common.BKDBIN: setIDs}}
	query := &meta.QueryCondition{
		Fields:    []string{common.BKSetIDField, common.BKSetNameField},
		Condition: cond,
	}
	sets, err := s.CoreAPI.CoreService().Instance().ReadInstance(ctx, header, common.BKInnerObjIDSet, query)
	if err != nil {
		blog.ErrorJSON("get set by condition: %s failed, err: %s, rid: %s", cond, err, rid)
		_ = resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: err})
		return
	}
	setMap := make(map[int64]string)
	for _, set := range sets.Data.Info {
		setID, err := set.Int64(common.BKSetIDField)
		if err != nil {
			blog.ErrorJSON("set %s id invalid, error: %s, rid: %s", set, err, rid)
			_ = resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: err})
			return
		}
		setName, err := set.String(common.BKSetNameField)
		if err != nil {
			blog.ErrorJSON("set %s name invalid, error: %s, rid: %s", set, err, rid)
			_ = resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: err})
			return
		}
		setMap[setID] = setName
	}

	cond = map[string]interface{}{common.BKModuleIDField: map[string]interface{}{common.BKDBIN: moduleIDs}}
	query = &meta.QueryCondition{
		Fields:    []string{common.BKModuleIDField, common.BKModuleNameField},
		Condition: cond,
	}
	modules, err := s.CoreAPI.CoreService().Instance().ReadInstance(ctx, header, common.BKInnerObjIDModule, query)
	if err != nil {
		blog.ErrorJSON("get module by condition: %s failed, err: %s, rid: %s", cond, err, rid)
		_ = resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: err})
		return
	}
	moduleMap := make(map[int64]string)
	for _, module := range modules.Data.Info {
		moduleID, err := module.Int64(common.BKModuleIDField)
		if err != nil {
			blog.ErrorJSON("module %s id invalid, error: %s, rid: %s", module, err, rid)
			_ = resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: err})
			return
		}
		moduleName, err := module.String(common.BKModuleNameField)
		if err != nil {
			blog.ErrorJSON("module %s name invalid, error: %s, rid: %s", module, err, rid)
			_ = resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: err})
			return
		}
		moduleMap[moduleID] = moduleName
	}

	// format the output
	hostTopos := meta.HostTopoResult{
		Count: hosts.Count,
	}
	for _, host := range hosts.Info {
		hostTopo := meta.HostTopo{
			Host: host,
		}
		topos := make([]meta.Topo, 0)
		hostID, _ := util.GetInt64ByInterface(host[common.BKHostIDField])
		if setModule, ok := relation[hostID]; ok {
			for setID, moduleIDs := range setModule {
				topo := meta.Topo{
					SetID:   setID,
					SetName: setMap[setID],
				}
				modules := make([]meta.Module, 0)
				for _, moduleID := range moduleIDs {
					module := meta.Module{
						ModuleID:   moduleID,
						ModuleName: moduleMap[moduleID],
					}
					modules = append(modules, module)
				}
				topo.Module = modules
				topos = append(topos, topo)
			}
		}
		hostTopo.Topo = topos
		hostTopos.Info = append(hostTopos.Info, hostTopo)
	}

	result := meta.NewSuccessResponse(hostTopos)
	_ = resp.WriteAsJson(result)
}
