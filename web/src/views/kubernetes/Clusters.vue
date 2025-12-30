<template>
  <div class="clusters-page">
    <el-card>
      <template #header>
        <h2>集群管理</h2>
      </template>
      <div class="placeholder">
        <el-empty description="集群管理功能开发中...">
          <el-button type="primary">创建集群</el-button>
        </el-empty>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
// 集群管理页面
</script>

<style scoped>
.clusters-page {
  padding: 20px;
}

.placeholder {
  padding: 100px 0;
  text-align: center;
}
</style>
