<template>
  <div class="endpoints-list">
    <div class="search-bar">
      <el-input v-model="searchName" placeholder="搜索 Endpoints 名称..." clearable class="search-input" @input="handleSearch">
        <template #prefix>
          <el-icon class="search-icon"><Search /></el-icon>
        </template>
      </el-input>

      <el-select v-model="filterNamespace" placeholder="命名空间" clearable @change="handleSearch" class="filter-select">
        <el-option label="全部" value="" />
        <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
      </el-select>

      <el-button @click="loadEndpoints">刷新</el-button>
    </div>

    <div class="table-wrapper">
      <el-table :data="filteredEndpoints" v-loading="loading" class="modern-table">
        <el-table-column label="名称" prop="name" min-width="180" />
        <el-table-column label="命名空间" prop="namespace" width="140" />
        <el-table-column label="端点" min-width="300">
          <template #default="{ row }">
            <div v-for="(subset, idx) in row.subsets" :key="idx" class="subset-item">
              <div class="subset-title">Subset {{ idx + 1 }}</div>
              <div class="subset-addresses">
                <div>就绪: {{ subset.addresses.length }}</div>
                <div>未就绪: {{ subset.notReadyAddresses.length }}</div>
                <div>端口: {{ subset.ports.map(p => `${p.port}/${p.protocol}`).join(', ') }}</div>
              </div>
            </div>
            <div v-if="!row.subsets.length">-</div>
          </template>
        </el-table-column>
        <el-table-column label="存活时间" prop="age" width="120" />
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button link @click="handleDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="detailDialogVisible" :title="`Endpoints 详情 - ${selectedEndpoint?.name}`" width="800px">
      <div v-if="selectedEndpoint">
        <div v-for="(subset, idx) in selectedEndpoint.subsets" :key="idx" class="detail-subset">
          <h4>Subset {{ idx + 1 }}</h4>
          <div><strong>就绪地址:</strong></div>
          <div v-for="(addr, i) in subset.addresses" :key="i" class="address-item">
            {{ addr.ip }} <span v-if="addr.targetRef">({{ addr.targetRef }})</span>
          </div>
          <div v-if="!subset.addresses.length">无</div>

          <div style="margin-top: 10px;"><strong>未就绪地址:</strong></div>
          <div v-for="(addr, i) in subset.notReadyAddresses" :key="i" class="address-item">
            {{ addr.ip }} <span v-if="addr.targetRef">({{ addr.targetRef }})</span>
          </div>
          <div v-if="!subset.notReadyAddresses.length">无</div>

          <div style="margin-top: 10px;"><strong>端口:</strong></div>
          <div>{{ subset.ports.map(p => `${p.name || '-'}: ${p.port}/${p.protocol}`).join(', ') || '-' }}</div>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import { getEndpoints, getEndpointsDetail, getNamespaces, type EndpointsInfo } from '@/api/kubernetes'

const props = defineProps<{
  clusterId?: number
  namespace?: string
}>()

const emit = defineEmits(['refresh'])

const loading = ref(false)
const endpointsList = ref<EndpointsInfo[]>([])
const namespaces = ref<any[]>([])
const searchName = ref('')
const filterNamespace = ref('')
const detailDialogVisible = ref(false)
const selectedEndpoint = ref<EndpointsInfo | null>(null)

const filteredEndpoints = computed(() => {
  let result = endpointsList.value
  if (searchName.value) {
    result = result.filter(e => e.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (filterNamespace.value) {
    result = result.filter(e => e.namespace === filterNamespace.value)
  }
  return result
})

const loadEndpoints = async () => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getEndpoints(props.clusterId, props.namespace || undefined)
    endpointsList.value = data || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 Endpoints 列表失败')
  } finally {
    loading.value = false
  }
}

const loadNamespaces = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
    console.error(error)
  }
}

const handleSearch = () => {
  // 本地过滤
}

const handleDetail = (endpoint: EndpointsInfo) => {
  selectedEndpoint.value = endpoint
  detailDialogVisible.value = true
}

watch(() => props.clusterId, () => {
  loadEndpoints()
  loadNamespaces()
})

watch(() => props.namespace, () => {
  filterNamespace.value = props.namespace || ''
  loadEndpoints()
})

onMounted(() => {
  loadEndpoints()
  loadNamespaces()
})
</script>

<style scoped>
.endpoints-list {
  width: 100%;
}

.search-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.search-input {
  width: 280px;
}

.filter-select {
  width: 180px;
}

.search-icon {
  color: #d4af37;
}

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.subset-item {
  margin-bottom: 8px;
}

.subset-title {
  font-size: 12px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.subset-addresses {
  font-size: 12px;
  color: #606266;
  line-height: 1.4;
}

.detail-subset {
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #eee;
}

.detail-subset:last-child {
  border-bottom: none;
}

.address-item {
  font-size: 13px;
  color: #606266;
  padding: 4px 0;
}
</style>
