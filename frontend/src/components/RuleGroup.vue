<template>
  <div class="rule-group" :style="{ borderLeft: isRoot ? 'none' : '2px solid #409EFF', paddingLeft: isRoot ? '0' : '15px', marginTop: '10px' }">
    <!-- Group Header -->
    <div style="display: flex; gap: 10px; align-items: center; margin-bottom: 10px;">
      <el-checkbox v-if="!isRoot" v-model="group.negate" border size="small" style="margin-right: 0">NOT</el-checkbox>
      <el-select v-model="group.conditionLogic" style="width: 90px;" size="small" v-if="group.isGroup || isRoot">
        <el-option label="AND" value="AND" />
        <el-option label="OR" value="OR" />
      </el-select>
      <el-button type="primary" plain size="small" @click="addCondition"><el-icon><Plus /></el-icon> Add Condition</el-button>
      <el-button type="success" plain size="small" @click="addGroup"><el-icon><Plus /></el-icon> Add Group</el-button>
      <el-button v-if="!isRoot" type="danger" plain size="small" @click="$emit('remove')"><el-icon><Delete /></el-icon> Remove Group</el-button>
    </div>

    <!-- Children -->
    <div v-for="(child, index) in group.conditions" :key="index" style="margin-bottom: 8px;">
      <template v-if="child.isGroup">
        <RuleGroup :group="child" :is-root="false" :raw-field-keys="rawFieldKeys" :fetch-suggestions="fetchSuggestions" @remove="removeChild(index)" />
      </template>
      <template v-else>
        <div style="display: flex; gap: 10px; align-items: center;">
          <el-checkbox v-model="child.negate" border size="small" style="margin-right: 0">NOT</el-checkbox>
          <el-select v-model="child.matchField" filterable allow-create placeholder="Field (Raw Data Key)" style="width: 200px">
            <el-option v-for="key in rawFieldKeys" :key="key" :label="key" :value="key" />
          </el-select>
          <el-select v-model="child.matchType" style="width: 120px">
            <el-option label="Equals" value="Equals" />
            <el-option label="Contains" value="Contains" />
            <el-option label="Regex" value="Regex" />
            <el-option label="> (Greater)" value=">" />
            <el-option label="< (Less)" value="<" />
          </el-select>
          <el-autocomplete
            v-model="child.matchValue"
            :fetch-suggestions="fetchSuggestions(child.matchField)"
            placeholder="Value (Type or Select)"
            style="flex: 1"
            clearable
          />
          <el-button type="danger" plain size="small" @click="removeChild(index)"><el-icon><Delete /></el-icon></el-button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus, Delete } from '@element-plus/icons-vue'

const props = defineProps<{
  group: any
  isRoot?: boolean
  rawFieldKeys: string[]
  fetchSuggestions: (field: string) => (queryString: string, cb: (data: any[]) => void) => void
}>()

const emit = defineEmits(['remove'])

const addCondition = () => {
  if (!props.group.conditions) {
    props.group.conditions = []
  }
  props.group.conditions.push({
    isGroup: false,
    matchField: '',
    matchType: 'Contains',
    matchValue: ''
  })
}

const addGroup = () => {
  if (!props.group.conditions) {
    props.group.conditions = []
  }
  props.group.conditions.push({
    isGroup: true,
    conditionLogic: 'AND',
    conditions: [
      { isGroup: false, matchField: '', matchType: 'Contains', matchValue: '' }
    ]
  })
}

const removeChild = (index: number | string) => {
  props.group.conditions.splice(Number(index), 1)
}
</script>

<style scoped>
.rule-group {
  transition: all 0.3s ease;
}
</style>
