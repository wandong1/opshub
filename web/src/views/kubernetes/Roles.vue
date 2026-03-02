<template>
  <div class="roles-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="page-title">角色管理</h2>
      <div class="header-actions">
        <a-select
          v-model="selectedClusterId"
          placeholder="请选择集群"
          style="width: 250px; margin-right: 12px;"
          @change="handleClusterChange"
        >
          <a-option
            v-for="cluster in clusters"
            :key="cluster.id"
            :label="cluster.alias || cluster.name"
            :value="cluster.id"
          />
        </a-select>
        <a-button class="black-button" @click="handleRefresh">
          <icon-refresh />
          刷新
        </a-button>
      </div>
    </div>

    <!-- 提示信息 -->
    <a-alert
      v-if="!selectedClusterId"
      title="请先选择一个集群"
      type="info"
      :closable="false"
      style="margin-bottom: 20px;"
    />

    <!-- 角色类型标签页 -->
    <a-tabs v-if="selectedClusterId" v-model:active-key="activeTab" class="role-tabs" @tab-change="handleTabChange">
      <a-tab-pane title="集群角色" key="cluster">
        <ClusterRoles :cluster-id="selectedClusterId" @role-click="handleRoleDetail" />
      </a-tab-pane>
      <a-tab-pane title="命名空间角色" key="namespace">
        <NamespaceRoles :cluster-id="selectedClusterId" @role-click="handleRoleDetail" />
      </a-tab-pane>
    </a-tabs>

    <!-- 角色详情对话框 -->
    <a-modal
      v-model="detailDialogVisible"
      :title="roleDetail.name"
      width="900px"
      destroy-on-close
    >
      <RoleDetail
        v-if="roleDetail && selectedClusterId"
        :cluster-id="selectedClusterId"
        :role="roleDetail"
        @close="detailDialogVisible = false"
      />
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import ClusterRoles from './components/ClusterRoles.vue'
import NamespaceRoles from './components/NamespaceRoles.vue'
import RoleDetail from './components/RoleDetail.vue'
import { getClusterList, type Cluster } from '@/api/kubernetes'

const activeTab = ref('cluster')
const detailDialogVisible = ref(false)
const roleDetail = ref<any>(null)
const selectedClusterId = ref<number | null>(null)
const clusters = ref<Cluster[]>([])

// 加载集群列表
const loadClusters = async () => {
  try {
    const list = await getClusterList()
    clusters.value = list

    // 如果有集群列表，默认选择第一个
    if (list.length > 0 && !selectedClusterId.value) {
      selectedClusterId.value = list[0].id
    }
  } catch (error) {
    // 错误处理
  }
}

const handleTabChange = () => {
  // 切换标签时可以刷新数据
}

const handleRefresh = () => {
  // 刷新当前标签页数据
  window.location.reload()
}

const handleClusterChange = () => {
  // 集群切换时，清空角色详情
  roleDetail.value = null
}

const handleRoleDetail = (role: any) => {
  roleDetail.value = role
  detailDialogVisible.value = true
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped lang="scss">
.roles-container {
  padding: 20px;

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    padding-bottom: 15px;
    border-bottom: 1px solid #e0e0e0;

    .page-title {
      margin: 0;
      font-size: 24px;
      font-weight: 500;
      color: #333;
    }

    .header-actions {
      display: flex;
      align-items: center;
    }
  }

  .black-button {
    background: linear-gradient(135deg, #2c3e50 0%, #000000 100%);
    color: #D4AF37;
    border: 1px solid rgba(212, 175, 55, 0.3);
    font-weight: 500;
    padding: 10px 20px;
    font-size: 14px;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.3s ease;

    &:hover {
      box-shadow: 0 4px 12px rgba(212, 175, 55, 0.4);
      transform: translateY(-1px);
    }

    &:active {
      transform: translateY(0);
    }
  }

  .role-tabs {
    :deep(.arco-tabs__header) {
      margin: 0 0 20px 0;
    }

    :deep(.arco-tabs__nav-wrap::after) {
      background-color: #e0e0e0;
    }
  }
}
</style>
