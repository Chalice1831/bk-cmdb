<template>
    <div class="table-layout" v-show="show">
        <div class="table-title" @click="localExpanded = !localExpanded">
            <cmdb-form-bool class="title-checkbox"
                :size="16"
                v-model="checked"
                @click.native.stop>
            </cmdb-form-bool>
            <i class="title-icon bk-icon icon-down-shape" v-if="localExpanded"></i>
            <i class="title-icon bk-icon icon-right-shape" v-else></i>
            <span class="title-label">{{instance.name}}</span>
            <cmdb-dot-menu class="instance-menu" @click.native.stop>
                <ul class="menu-list">
                    <li class="menu-item"
                        v-for="(menu, index) in instanceMenu"
                        :key="index">
                        <button class="menu-button"
                            @click="menu.handler">
                            {{menu.name}}
                        </button>
                    </li>
                </ul>
            </cmdb-dot-menu>
            <div class="right-content fr">
                <div class="instance-label clearfix" @click.stop v-if="currentView === 'label'">
                    <div class="label-list fl">
                        <div class="label-item" :title="`${label.key}：${label.value}`" :key="index" v-for="(label, index) in labelShowList">
                            <span>{{label.key}}</span>
                            <span>:</span>
                            <span>{{label.value}}</span>
                        </div>
                        <bk-popover class="label-item label-tips"
                            v-if="labelTipsList.length"
                            theme="light label-tips"
                            :width="290"
                            placement="bottom-end">
                            <span>...</span>
                            <div class="tips-label-list" slot="content">
                                <span class="label-item" :title="`${label.key}：${label.value}`" :key="index" v-for="(label, index) in labelTipsList">
                                    <span>{{label.key}}</span>
                                    <span>:</span>
                                    <span>{{label.value}}</span>
                                </span>
                            </div>
                        </bk-popover>
                    </div>
                </div>
                <span class="topology-path" v-else
                    v-bk-tooltips="pathToolTips"
                    @click.stop="goTopologyInstance">
                    {{topologyPath}}
                </span>
            </div>
        </div>
        <bk-table
            v-show="localExpanded"
            v-bkloading="{ isLoading: $loading(Object.values(requestId)) }"
            :data="flattenList">
            <bk-table-column v-for="column in header"
                :key="column.id"
                :prop="column.id"
                :label="column.name">
            </bk-table-column>
        </bk-table>
    </div>
</template>

<script>
    export default {
        props: {
            instance: {
                type: Object,
                required: true
            },
            expanded: Boolean,
            currentView: {
                type: String,
                default: 'label'
            }
        },
        data () {
            return {
                tipsLabel: {
                    show: false,
                    id: null
                },
                show: true,
                checked: false,
                localExpanded: this.expanded,
                properties: [],
                header: [],
                list: [],
                pathToolTips: {
                    content: this.$t('BusinessTopology["跳转服务拓扑"]'),
                    placement: 'top'
                }
            }
        },
        computed: {
            isModuleNode () {
                const node = this.$store.state.businessTopology.selectedNode
                return node && node.data.bk_obj_id === 'module'
            },
            withTemplate () {
                return this.isModuleNode && !!this.instance.service_template_id
            },
            instanceMenu () {
                const menu = [{
                    name: this.$t('Common["删除"]'),
                    handler: this.handleDeleteInstance
                }]
                return menu
            },
            module () {
                return this.$store.state.businessTopology.selectedNodeInstance
            },
            flattenList () {
                return this.$tools.flattenList(this.properties, this.list.map(data => data.property))
            },
            requestId () {
                return {
                    processList: `get_service_instance_${this.instance.id}_processes`,
                    properties: 'get_service_process_properties',
                    deleteProcess: 'delete_service_process'
                }
            },
            labelList () {
                const list = []
                const labels = this.instance.labels
                labels && Object.keys(labels).forEach(key => {
                    list.push({
                        id: this.instance.id,
                        key: key,
                        value: labels[key]
                    })
                })
                return list
            },
            labelShowList () {
                return this.labelList.slice(0, 3)
            },
            labelTipsList () {
                return this.labelList.slice(3)
            },
            topologyPath () {
                const pathArr = this.$tools.clone(this.instance.topo_path).reverse()
                const path = pathArr.map(path => path.InstanceName).join(' / ')
                return path
            }
        },
        watch: {
            localExpanded (expanded) {
                if (expanded) {
                    this.getServiceProcessList()
                }
            },
            checked (checked) {
                this.$emit('check-change', checked, this.instance)
            }
        },
        async created () {
            await this.getProcessProperties()
            if (this.expanded) {
                this.getServiceProcessList()
            }
        },
        methods: {
            async getProcessProperties () {
                const action = 'objectModelProperty/searchObjectAttribute'
                const properties = await this.$store.dispatch(action, {
                    params: {
                        bk_obj_id: 'process',
                        bk_supplier_account: this.$store.getters.supplierAccount
                    },
                    config: {
                        requestId: this.requestId.properties,
                        fromCache: true
                    }
                })
                this.properties = properties
                this.setHeader()
            },
            async getServiceProcessList () {
                try {
                    this.list = await this.$store.dispatch('processInstance/getServiceInstanceProcesses', {
                        params: this.$injectMetadata({
                            service_instance_id: this.instance.id
                        }),
                        config: {
                            requestId: this.requestId.processList
                        }
                    })
                } catch (e) {
                    this.list = []
                    console.error(e)
                }
            },
            setHeader () {
                const display = [
                    'bk_func_name',
                    'bk_process_name',
                    'bk_start_param_regex',
                    'bind_ip',
                    'port',
                    'work_path'
                ]
                const header = display.map(id => {
                    const property = this.properties.find(property => property.bk_property_id === id) || {}
                    return {
                        id: property.bk_property_id,
                        name: property.bk_property_name
                    }
                })
                this.header = header
            },
            handleDeleteInstance () {
                this.$bkInfo({
                    title: this.$t('BusinessTopology["确认删除实例"]'),
                    subTitle: this.$t('BusinessTopology["即将删除实例"]', { name: this.instance.name }),
                    confirmFn: async () => {
                        try {
                            await this.$store.dispatch('serviceInstance/deleteServiceInstance', {
                                config: {
                                    data: this.$injectMetadata({
                                        service_instance_ids: [this.instance.id]
                                    }),
                                    requestId: this.requestId.deleteProcess
                                }
                            })
                            this.$emit('delete-instance', this.instance.id)
                        } catch (e) {
                            console.error(e)
                        }
                    }
                })
            },
            goTopologyInstance () {
                this.$router.replace({
                    name: 'topology',
                    query: {
                        module: this.instance.bk_module_id,
                        instanceName: this.instance.name
                    }
                })
            }
        }
    }
</script>

<style lang="scss" scoped>
    .table-layout {
        padding: 0 0 12px 0;
    }
    .table-title {
        height: 40px;
        padding: 0 11px;
        line-height: 40px;
        border-radius: 2px 2px 0 0;
        background-color: #DCDEE5;
        overflow: hidden;
        cursor: pointer;
        .title-icon {
            font-size: 12px;
            color: #63656E;
            @include inlineBlock;
        }
        .title-label {
            font-size: 14px;
            color: #313238;
            @include inlineBlock;
        }
        .topology-path {
            font-size: 12px;
            color: #63656e;
            height: 20px;
            line-height: 20px;
            padding: 0 6px;
            outline: none;
            @include inlineBlock;
            &:hover {
                color: #3a84ff;
            }
        }
    }
    .menu-list {
        min-width: 74px;
        padding: 6px 0;
        .menu-item {
            .menu-button {
                display: block;
                width: 100%;
                height: 32px;
                padding: 0 13px;
                line-height: 32px;
                outline: 0;
                border: none;
                text-align: left;
                color: #63656E;
                font-size: 12px;
                background-color: #fff;
                &:hover {
                    background-color: #E1ECFF;
                    color: #3A84FF;
                }
            }
        }
    }
    .instance-label {
        @include inlineBlock;
        font-size: 12px;
        .icon-cc-label {
            color: #979ba5;
            font-size: 16px;
        }
        .label-list {
            padding-left: 4px;
            line-height: 38px;
            font-size: 0;
            .label-item {
                @include inlineBlock;
                font-size: 12px;
                height: 20px;
                line-height: 20px;
                margin-right: 4px;
                padding: 0 6px;
                color: #979ba5;
                background-color: #fafbfd;
                border-radius: 2px;
                &>span {
                    @include ellipsis;
                    display: inline-block;
                    max-width: 54px;
                }
            }
            .label-tips {
                padding: 0;
                /deep/.bk-tooltip-ref {
                    padding: 0 6px;
                    span {
                        line-height: 16px;
                        display: inline-block;
                        vertical-align: top;
                    }
                }
                &:hover {
                    background-color: #e1ecff;
                }
            }
        }
    }
</style>

<style lang="scss">
    .tips-label-list {
        .label-item {
            @include inlineBlock;
            font-size: 12px;
            height: 20px;
            line-height: 18px;
            margin: 5px 2px;
            padding: 0 6px;
            color: #979ba5;
            background-color: #fafbfd;
            border: 1px solid #dcdee5;
            border-radius: 2px;
            &>span {
                @include ellipsis;
                display: inline-block;
                max-width: 54px;
            }
        }
    }
    .tippy-tooltip.label-tips-theme {
        padding: 8px 6px !important;
    }
</style>