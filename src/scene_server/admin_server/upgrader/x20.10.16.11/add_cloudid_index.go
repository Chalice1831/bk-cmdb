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

package x20_10_16_11

import (
	"context"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/scene_server/admin_server/upgrader"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/util"
)

func addCloudIDIndex(ctx context.Context, db dal.RDB, conf *upgrader.Config) error {

	idx := dal.Index{
		Keys: map[string]int32{
			common.BKCloudIDField: 1,
		},
		Name:       "bk_cloud_id_1",
		Unique:     false,
		Background: true,
	}

	if err := db.Table(common.BKTableNameBaseHost).CreateIndex(ctx, idx); err != nil {
		if !util.IsDuplicatedIndexErr(err) {
			blog.Errorf("CreateIndex failed, idx:%#v, err: %+v", idx, err)
			return err
		}
	}

	if err := db.Table(common.BKTableNameBasePlat).CreateIndex(ctx, idx); err != nil {
		if !util.IsDuplicatedIndexErr(err) {
			blog.Errorf("CreateIndex failed, idx:%#v, err: %+v", idx, err)
			return err
		}
	}

	return nil
}
