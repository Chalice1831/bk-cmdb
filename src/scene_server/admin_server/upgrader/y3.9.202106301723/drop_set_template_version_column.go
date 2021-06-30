package y3_9_202106301723

import (
	"configcenter/src/common/util"
	"context"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/scene_server/admin_server/upgrader"
	"configcenter/src/storage/dal"
)

const step = 5000

func dropVersionColumn(ctx context.Context, db dal.RDB, conf *upgrader.Config) error {
	existsVersionFilter := map[string]interface{}{
		"version": map[string]interface{}{
			common.BKDBExists: true,
		},
	}

	for {
		setTpls := make([]map[string]interface{}, 0)
		err := db.Table(common.BKTableNameSetTemplate).Find(existsVersionFilter).Fields(common.BKFieldID).
			Start(0).Limit(step).All(ctx, &setTpls)
		if err != nil {
			blog.Errorf("count table %s failed, err: %s", common.BKTableNameSetTemplate, err.Error())
			return err
		}

		if len(setTpls) == 0 {
			break
		}

		setTplIDs := make([]int64, len(setTpls))
		for index, setTpl := range setTpls {
			setTplID, err := util.GetInt64ByInterface(setTpl[common.BKFieldID])
			if err != nil {
				blog.Errorf("get set template id failed, set: %+v, err: %v", setTpl, err)
				return err
			}
			setTplIDs[index] = setTplID
		}

		filter := map[string]interface{}{
			common.BKFieldID: map[string]interface{}{common.BKDBIN: setTplIDs},
		}
		if err := db.Table(common.BKTableNameSetTemplate).DropColumns(ctx, filter, []string{"version"}); err != nil {
			blog.Errorf("drop column failed, field:%s, err:%v", "version", err)
			return err
		}
	}

	return nil
}

func dropSetTplVersionColumn(ctx context.Context, db dal.RDB, conf *upgrader.Config) error {
	existsVersionFilter := map[string]interface{}{
		common.BKSetTemplateVersionField: map[string]interface{}{
			common.BKDBExists: true,
		},
	}

	for {
		sets := make([]map[string]interface{}, 0)
		err := db.Table(common.BKTableNameBaseSet).Find(existsVersionFilter).Fields(common.BKSetIDField).
			Start(0).Limit(step).All(ctx, &sets)
		if err != nil {
			blog.Errorf("count table %s failed, err: %s", common.BKTableNameBaseSet, err.Error())
			return err
		}

		if len(sets) == 0 {
			break
		}

		setIDs := make([]int64, len(sets))
		for index, set := range sets {
			setID, err := util.GetInt64ByInterface(set[common.BKSetIDField])
			if err != nil {
				blog.Errorf("get set id failed, set: %+v, err: %v", set, err)
				return err
			}
			setIDs[index] = setID
		}

		filter := map[string]interface{}{
			common.BKSetIDField: map[string]interface{}{common.BKDBIN: setIDs},
		}
		if err := db.Table(common.BKTableNameBaseSet).
			DropColumns(ctx, filter, []string{common.BKSetTemplateVersionField}); err != nil {
			blog.Errorf("drop column failed, field:%s, err:%v", common.BKSetTemplateVersionField, err)
			return err
		}
	}

	return nil
}
