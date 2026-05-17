<template>
  <div 
    class="rule-node-wrapper"
    draggable="true"
    @dragstart.stop="onDragStart"
    @dragover.stop.prevent="onDragOver"
    @dragleave="onDragLeave"
    @drop.stop="onDrop"
    @dragend="onDragEnd"
    :class="{ 
      'drop-left': dropZone === 'left', 
      'drop-right': dropZone === 'right',
      'is-group': rule.isRuleGroup,
      'is-rule': !rule.isRuleGroup
    }"
  >
    <!-- Group -->
    <div v-if="rule.isRuleGroup" class="rule-group-box">
      <div class="group-header">
        <div class="left" style="cursor: pointer" @click="collapsed = !collapsed">
          <el-icon :class="{ 'rotate-icon': collapsed }"><ArrowDown /></el-icon>
          <strong><el-icon><Folder /></el-icon> {{ rule.name || 'Untitled Group' }}</strong>
          <el-tag size="small" type="" v-if="rule.rules && rule.rules.length">{{ rule.rules.length }} rules</el-tag>
          <div class="platforms">
            <el-tag size="small" type="info" v-if="!rule.excludeAlipay">Alipay</el-tag>
            <el-tag size="small" type="success" v-if="!rule.excludeWechat">WeChat</el-tag>
          </div>
        </div>
        <div class="actions">
          <el-button size="small" :icon="ArrowUp" circle @click="$emit('move-up')" :disabled="isFirst" />
          <el-button size="small" :icon="ArrowDown" circle @click="$emit('move-down')" :disabled="isLast" />
          <el-button size="small" @click="$emit('edit')">Edit Group</el-button>
          <el-tooltip :disabled="!rule.rules || !rule.rules.length" content="Please remove all rules inside this group first" placement="top">
            <el-button size="small" type="danger" plain @click="$emit('remove')" :disabled="rule.rules && rule.rules.length > 0">Delete Group</el-button>
          </el-tooltip>
        </div>
      </div>
      <div v-if="rule.conditions && rule.conditions.length" class="group-condition" @click.stop>
        <el-tag size="small" type="warning">IF</el-tag>
        <RuleDisplay :group="rule" />
      </div>

      <div class="group-children" v-show="!collapsed">
        <RuleNode 
          v-for="(child, idx) in rule.rules" 
          :key="idx" 
          :rule="child"
          :path="[...path, Number(idx)]"
          :is-first="idx === 0"
          :is-last="idx === (rule.rules ? rule.rules.length - 1 : 0)"
          @edit="$emit('edit-child', {rule: child, parent: rule, index: idx})"
          @remove="() => removeChild(Number(idx))"
          @move-up="() => moveChildUp(Number(idx))"
          @move-down="() => moveChildDown(Number(idx))"
          @edit-child="(payload) => $emit('edit-child', payload)"
          @add-rule="(parent) => $emit('add-rule', parent)"
          @add-group="(parent) => $emit('add-group', parent)"
        />
        <div class="add-buttons">
          <el-button size="small" type="primary" plain @click="$emit('add-rule', rule)">+ Add Rule Here</el-button>
          <el-button size="small" type="success" plain @click="$emit('add-group', rule)">+ Add Sub-Group</el-button>
        </div>
      </div>
    </div>

    <!-- Rule Card -->
    <el-card v-else class="rule-card" shadow="hover">
      <div class="rule-header">
        <div class="left">
          <strong><el-icon><Document /></el-icon> {{ rule.name || 'Untitled Rule' }}</strong>
          <div class="platforms">
            <el-tag size="small" type="info" v-if="!rule.excludeAlipay">Alipay</el-tag>
            <el-tag size="small" type="success" v-if="!rule.excludeWechat">WeChat</el-tag>
          </div>
        </div>
        <div class="actions">
          <el-button size="small" :icon="ArrowUp" circle @click="$emit('move-up')" :disabled="isFirst" />
          <el-button size="small" :icon="ArrowDown" circle @click="$emit('move-down')" :disabled="isLast" />
          <el-button size="small" type="primary" plain @click="$emit('edit')">Edit</el-button>
          <el-button size="small" type="danger" plain @click="$emit('remove')">Delete</el-button>
        </div>
      </div>
      <div class="rule-body">
        <div class="rule-if">
          <el-tag size="small" type="warning">IF</el-tag> 
          <RuleDisplay :group="rule" />
        </div>
        <div class="rule-then">
          <el-tag size="small" type="success">THEN</el-tag>
          <span v-if="rule.ignore" class="action-item strike">Skip Transaction</span>
          <span v-if="rule.targetType" class="action-item">Type: {{ rule.targetType }}</span>
          <span v-if="rule.targetCategory" class="action-item">Category: {{ rule.targetCategory }}</span>
          <span v-if="rule.targetAsset" class="action-item">Asset: {{ rule.targetAsset }}</span>
          <span v-if="rule.targetOpposing" class="action-item">Opposing: {{ rule.targetOpposing }}</span>
          <span v-if="rule.swapAccounts" class="action-item">Swap Accounts</span>
          <span v-if="rule.targetDescription" class="action-item">Desc: {{ rule.targetDescription }}</span>
          <span v-if="rule.targetNotes" class="action-item">Notes: {{ rule.targetNotes }}</span>
          <span v-if="rule.targetTags" class="action-item">Tags: {{ rule.targetTags }}</span>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { inject, ref } from 'vue'
import { Folder, Document, ArrowUp, ArrowDown } from '@element-plus/icons-vue'
import RuleDisplay from './RuleDisplay.vue'

const props = defineProps<{
  rule: any
  path: number[]
  isFirst: boolean
  isLast: boolean
}>()

const emit = defineEmits(['edit', 'remove', 'move-up', 'move-down', 'add-rule', 'add-group', 'edit-child'])

const dragState = inject<any>('dragState')
const handleTreeDrop = inject<any>('handleTreeDrop')

const collapsed = ref(false)
const dropZone = ref('')

const onDragStart = (e: DragEvent) => {
  if (dragState) {
    dragState.sourcePath = props.path
  }
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', JSON.stringify(props.path))
  }
}

const onDragOver = (e: DragEvent) => {
  if (!dragState || !dragState.sourcePath) return
  if (JSON.stringify(dragState.sourcePath) === JSON.stringify(props.path)) return

  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  const isRight = e.clientX - rect.left > rect.width / 2

  if (isRight && props.rule.isRuleGroup) {
    dropZone.value = 'right'
  } else {
    dropZone.value = 'left'
  }
}

const onDragLeave = () => {
  dropZone.value = ''
}

const onDrop = () => {
  if (dropZone.value && dragState && dragState.sourcePath && handleTreeDrop) {
    handleTreeDrop(dragState.sourcePath, props.path, dropZone.value)
  }
  dropZone.value = ''
  if (dragState) dragState.sourcePath = null
}

const onDragEnd = () => {
  dropZone.value = ''
  if (dragState) dragState.sourcePath = null
}

const removeChild = (idx: number) => {
  props.rule.rules.splice(idx, 1)
}

const moveChildUp = (idx: number) => {
  if (idx > 0) {
    const temp = props.rule.rules[idx]
    props.rule.rules[idx] = props.rule.rules[idx - 1]
    props.rule.rules[idx - 1] = temp
  }
}

const moveChildDown = (idx: number) => {
  if (idx < props.rule.rules.length - 1) {
    const temp = props.rule.rules[idx]
    props.rule.rules[idx] = props.rule.rules[idx + 1]
    props.rule.rules[idx + 1] = temp
  }
}
</script>

<style scoped>
.rule-node-wrapper {
  margin-bottom: 12px;
  transition: all 0.2s;
}
.rule-node-wrapper.is-group {
  flex: 1 1 100%;
}
.rule-node-wrapper.is-rule {
  flex: 1 1 350px;
  min-width: 0;
}
.drop-left {
  border-top: 4px solid #409EFF;
  padding-top: 4px;
}
.drop-right > .rule-group-box {
  background-color: #ecf5ff;
  border-color: #409EFF;
}
.rule-group-box {
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  background-color: #f8f9fa;
  padding: 10px;
  transition: all 0.2s;
}
.group-condition {
  font-size: 13px;
  padding: 6px 10px;
  margin-bottom: 8px;
  background: #fdf6ec;
  border: 1px solid #faecd8;
  border-radius: 4px;
}
.rotate-icon {
  transform: rotate(-90deg);
  transition: transform 0.2s;
}
.group-header {
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 10px;
}
.group-children {
  margin-left: 10px;
  padding-left: 15px;
  border-left: 2px dashed #c0c4cc;
  display: flex;
  flex-wrap: wrap;
  gap: 15px;
}
.rule-card {
  margin-bottom: 0;
  height: 100%;
}
.rule-header {
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #eee;
}
.actions {
  display: flex;
  justify-content: flex-end;
  width: 100%;
}
.left {
  display: flex;
  align-items: center;
  gap: 10px;
}
.platforms {
  display: flex;
  gap: 5px;
}
.rule-body {
  font-size: 13px;
}
.rule-if, .rule-then {
  margin-bottom: 8px;
}
.action-item {
  display: inline-block;
  background: #f0f9eb;
  border: 1px solid #e1f3d8;
  color: #67c23a;
  padding: 2px 8px;
  border-radius: 4px;
  margin-right: 6px;
  margin-top: 4px;
}
.strike {
  text-decoration: line-through;
  color: #f56c6c;
  background: #fef0f0;
  border-color: #fde2e2;
}
.add-buttons {
  flex: 1 1 100%;
  margin-top: 10px;
  display: flex;
  gap: 10px;
}
</style>
