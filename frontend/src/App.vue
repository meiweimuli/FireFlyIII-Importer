<template>
  <div class="app-container">
    <h1 class="title">FireFly III Importer</h1>

    <el-tabs v-model="activeTab" class="glass-card">
      <el-tab-pane label="Import Data" name="import">
        <div class="upload-section">
          <el-upload
            class="upload-demo"
            drag
            action="http://localhost:8080/api/upload"
            :on-success="handleUploadSuccess"
            :on-error="handleUploadError"
            :show-file-list="false"
          >
            <el-icon class="el-icon--upload"><upload-filled /></el-icon>
            <div class="el-upload__text">
              Drop Alipay CSV or WeChat XLSX here, or <em>click to upload</em>
            </div>
          </el-upload>
        </div>

        <div v-if="transactions.length > 0" class="preview-section">
          <div class="actions" style="display: flex; gap: 10px; align-items: center">
            <el-button type="primary" size="large" @click="submitImport(false)" :loading="importing">
              Start Import ({{ validTransactionsCount }} items)
            </el-button>
            <el-button v-if="selectedTransactions.length > 0" type="warning" size="large" @click="submitImport(true)" :loading="importing">
              Import Selected ({{ selectedTransactions.length }} items)
            </el-button>

            <!-- Import Progress Dialog -->
            <el-dialog v-model="importProgressVisible" title="Import Progress" width="700px" :close-on-click-modal="false" :close-on-press-escape="false" :show-close="!importing" :draggable="false" append-to-body>
              <el-progress :percentage="importProgress.percentage" :status="importProgress.status" :stroke-width="20" style="margin-bottom: 20px" />
              <div style="margin-bottom: 10px; font-size: 14px; color: #606266;">
                Processed: {{ importProgress.current }} / {{ importProgress.total }}
                <span v-if="importProgress.success > 0" style="color: #67c23a; margin-left: 15px;">✓ {{ importProgress.success }}</span>
                <span v-if="importProgress.failed > 0" style="color: #f56c6c; margin-left: 15px;">✗ {{ importProgress.failed }}</span>
                <span v-if="importProgress.skipped > 0" style="color: #e6a23c; margin-left: 15px;">⊘ {{ importProgress.skipped }}</span>
              </div>
              <el-table :data="importResults" height="350" size="small">
                <el-table-column label="#" width="50" type="index" />
                <el-table-column label="Description" prop="description" min-width="200" show-overflow-tooltip />
                <el-table-column label="Status" width="100">
                  <template #default="scope">
                    <el-tag :type="scope.row.status === 'success' ? 'success' : scope.row.status === 'error' ? 'danger' : 'warning'" size="small">
                      {{ scope.row.status }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="Message" prop="message" min-width="250" show-overflow-tooltip />
              </el-table>
              <template #footer>
                <el-button v-if="importing" type="danger" @click="stopImport">Stop Import</el-button>
                <el-button v-else @click="importProgressVisible = false">Close</el-button>
              </template>
            </el-dialog>
            <el-button @click="evaluateTransactions">
              Re-Apply Rules
            </el-button>
            <el-button type="warning" plain @click="resetSelectedTransactions" :disabled="selectedTransactions.length === 0">
              Reset Selected
            </el-button>
            <el-button type="primary" plain @click="openBulkEditor" :disabled="selectedTransactions.length === 0">
              Bulk Edit ({{ selectedTransactions.length }})
            </el-button>
            <el-button type="danger" plain @click="resetAndEvaluateTransactions" :disabled="originalTransactions.length === 0">
              Reset All
            </el-button>
            <div style="flex: 1"></div>
            <el-input v-model="searchQuery" placeholder="Search any keyword or raw data..." style="width: 300px" clearable />
          </div>

          <el-table ref="txTable" :data="filteredTransactions" style="width: 100%" height="500" :row-class-name="tableRowClassName" @selection-change="handleSelectionChange" @cell-click="handleCellClick">
            <el-table-column type="selection" width="55" fixed="left" />

            <el-table-column prop="date" label="Date" width="160">
              <template #default="scope">
                {{ scope.row.date ? new Date(scope.row.date).toLocaleString() : '' }}
              </template>
            </el-table-column>
            <el-table-column prop="type" label="Type" width="100"
              :filters="[{text: 'Withdrawal', value: 'withdrawal'}, {text: 'Deposit', value: 'deposit'}, {text: 'Transfer', value: 'transfer'}]"
              :filter-method="filterType">
              <template #default="scope">
                <el-tag :type="scope.row.type === 'deposit' ? 'success' : scope.row.type === 'transfer' ? 'warning' : 'danger'">
                  {{ scope.row.type }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="amount" label="Amount" width="100" />
            <el-table-column prop="paymentMethod" label="Payment Method" width="130" show-overflow-tooltip />
            <el-table-column prop="counterparty" label="Counterparty" width="150" show-overflow-tooltip />
            <el-table-column prop="description" label="Description" min-width="180">
              <template #default="scope">
                <el-popover placement="right" title="Raw Original Data" :width="450" trigger="hover">
                  <template #reference>
                    <div style="display: flex; align-items: center; gap: 5px; cursor: pointer; width: 100%;">
                      <el-icon color="#909399" style="flex-shrink: 0;"><InfoFilled /></el-icon>
                      <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1;">
                        {{ scope.row.description || '(Empty)' }}
                      </span>
                    </div>
                  </template>
                  <el-descriptions :column="1" border size="small" direction="horizontal">
                    <el-descriptions-item v-for="(value, key) in parseRawData(scope.row.rawData)" :key="key" :label="key" label-width="120px" label-align="right">
                      {{ value }}
                    </el-descriptions-item>
                  </el-descriptions>
                </el-popover>
              </template>
            </el-table-column>
            <el-table-column prop="notes" label="Notes" min-width="150" show-overflow-tooltip />
            <el-table-column prop="category" label="Category" width="160"
              :filters="[{text: '未确定 (Empty)', value: 'empty'}, {text: '已确定 (Mapped)', value: 'mapped'}]"
              :filter-method="filterCategory">
            </el-table-column>
            <el-table-column label="Accounts" width="260"
              :filters="[{text: '缺少资金账户', value: 'missing_asset'}, {text: '缺少相对账户', value: 'missing_opposing'}, {text: '默认资金账户 (Default Asset)', value: 'default_asset'}, {text: '默认对方账户 (Default Opposing)', value: 'default_opposing'}]"
              :filter-method="filterAccounts">
              <template #default="scope">
                <div style="margin-bottom: 4px; font-size: 0.9em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">
                  <span style="color: #666; font-weight: 500;">Asset:</span> {{ scope.row.assetAccount || '(Empty)' }}
                  <el-tag v-if="scope.row.assetAccount === config.defaultAssetAccount && config.defaultAssetAccount" size="small" type="info" style="margin-left: 4px">Default</el-tag>
                </div>
                <div style="font-size: 0.9em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">
                  <span style="color: #666; font-weight: 500;">Opposing:</span> {{ scope.row.opposingAccount || '(Empty)' }}
                  <el-tag v-if="scope.row.opposingAccount === config.defaultOpposingAccount && config.defaultOpposingAccount" size="small" type="info" style="margin-left: 4px">Default</el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="Tags" width="120">
              <template #default="scope">
                <el-tag v-for="tag in scope.row.tags" :key="tag" size="small" style="margin-right:4px; margin-bottom: 4px;">{{ tag }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="Status" width="100"
              :filters="[{text: '正常 (Import)', value: false}, {text: '跳过 (Skip)', value: true}]"
              :filter-method="filterStatus">
              <template #default="scope">
                <el-switch v-model="scope.row.ignore" inline-prompt active-text="Skip" inactive-text="Import" />
              </template>
            </el-table-column>
            <el-table-column label="Actions" width="200" fixed="right">
              <template #default="scope">
                <el-button size="small" type="primary" plain @click="openTxEditor(scope.row)">Edit</el-button>
                <el-button size="small" type="success" plain @click="createRuleFromRow(scope.row)">Rule</el-button>
                <el-button size="small" type="warning" plain @click="resetRow(scope.row)">Reset</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="Settings" name="settings">
        <el-form :model="config" label-width="200px">
          <el-form-item label="Firefly URL">
            <el-input v-model="config.fireflyUrl" placeholder="http://localhost:80" />
          </el-form-item>
          <el-form-item label="Personal Token">
            <el-input v-model="config.fireflyToken" type="password" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="info" @click="syncFireflyData" :loading="syncing" icon="Refresh">Sync FireFly III Data</el-button>
          </el-form-item>
          
          <el-divider>Global Settings</el-divider>
          <el-form-item label="Global Tags (Comma separated)">
            <el-input v-model="config.globalTags" placeholder="e.g. imported, wechat" />
          </el-form-item>
          <el-form-item label="Deduplicate by External ID">
            <el-switch v-model="config.deduplicateByExternalId" active-text="Yes" inactive-text="No" />
            <span style="margin-left: 10px; color: #909399; font-size: 13px;">Skip transactions with duplicate external IDs during import</span>
          </el-form-item>

          <el-divider>Default Accounts</el-divider>
          <el-form-item label="Default Asset Account">
            <el-select v-model="config.defaultAssetAccount" filterable allow-create default-first-option @change="updateConfigAssetAcc">
              <el-option label="-- 未确定 (Leave Empty) --" value="" />
              <el-option v-for="acc in assetAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
            </el-select>
          </el-form-item>
          <el-form-item label="Default Opposing Account">
            <el-select v-model="config.defaultOpposingAccount" filterable allow-create default-first-option @change="updateConfigOpposingAcc">
              <el-option label="-- 未确定 (Leave Empty) --" value="" />
              <el-option-group label="Expense (支出)">
                <el-option v-for="acc in expenseAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
              </el-option-group>
              <el-option-group label="Revenue (收入)">
                <el-option v-for="acc in revenueAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
              </el-option-group>
              <el-option-group label="Asset/Liability (转账)">
                <el-option v-for="acc in assetAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
              </el-option-group>
            </el-select>
          </el-form-item>

          <el-divider>External ID Fields</el-divider>
          <el-form-item label="Alipay ID Field">
            <el-input v-model="config.alipayExternalIdField" placeholder="Default: 交易订单号" />
          </el-form-item>
          <el-form-item label="WeChat ID Field">
            <el-input v-model="config.wechatExternalIdField" placeholder="Default: 交易单号" />
          </el-form-item>
          <div style="margin-top: 20px; display: flex; gap: 10px; align-items: center;">
            <el-button type="success" size="large" @click="saveConfig" :loading="savingConfig">Save All Settings</el-button>
            <el-button type="info" plain @click="exportConfig">Export Config</el-button>
            <el-upload
              :show-file-list="false"
              accept=".json"
              :before-upload="importConfig"
              style="display: inline-block;"
            >
              <el-button type="warning" plain>Import Config</el-button>
            </el-upload>
          </div>

          <el-divider>
            Advanced Mapping Rules
            <el-button type="primary" size="small" @click="openRuleEditor(null, null, -1)" style="margin-left: 20px;">Add New Rule</el-button>
            <el-button type="success" size="small" @click="addGroupTo(null)" style="margin-left: 10px;">Add New Group</el-button>
            <el-button type="warning" size="small" @click="loadPresets" style="margin-left: 10px;">Load Default China Presets</el-button>
          </el-divider>
          
          <div class="rules-list">
            <RuleNode 
              v-for="(rule, index) in config.mappingRules" 
              :key="index" 
              :rule="rule"
              :path="[index]"
              :is-first="index === 0"
              :is-last="index === (config.mappingRules ? config.mappingRules.length - 1 : 0)"
              @edit="openRuleEditor(rule, null, index)"
              @remove="() => removeRootRule(index)"
              @move-up="() => moveRootRuleUp(index)"
              @move-down="() => moveRootRuleDown(index)"
              @add-rule="openRuleEditor(null, rule, -1)"
              @add-group="addGroupTo(rule)"
              @edit-child="handleEditChild"
            />
          </div>

        </el-form>
      </el-tab-pane>
    </el-tabs>

    <!-- Rule Editor Dialog -->
    <el-dialog v-model="ruleDialogVisible" :title="editingRule.isRuleGroup ? 'Edit Rule Group' : 'Edit Mapping Rule'" width="700px">
      <el-form :model="editingRule" label-width="140px">
        <el-form-item label="Name">
          <el-input v-model="editingRule.name" placeholder="e.g. Meituan Food" />
        </el-form-item>

        <el-form-item label="Apply To Platforms">
          <el-checkbox :model-value="!editingRule.excludeAlipay" @update:model-value="(v: boolean) => editingRule.excludeAlipay = !v">Alipay</el-checkbox>
          <el-checkbox :model-value="!editingRule.excludeWechat" @update:model-value="(v: boolean) => editingRule.excludeWechat = !v">WeChat</el-checkbox>
        </el-form-item>

        <h4 v-if="editingRule.isRuleGroup">Trigger Conditions (Optional — leave empty to always execute children)</h4>
        <h4 v-else>1. Trigger Conditions (IF)</h4>
        <RuleGroup :group="editingRule" :is-root="true" :raw-field-keys="rawFieldKeys" :fetch-suggestions="fetchSuggestions" />

        <template v-if="!editingRule.isRuleGroup">
        <h4>2. Action (THEN)</h4>
        <el-form-item label="Set Ignore Flag">
          <div style="display: flex; gap: 10px; width: 100%; align-items: center">
            <el-checkbox v-model="editingRule.modifyIgnore" />
            <el-switch v-model="editingRule.ignore" active-text="Skip (Ignore)" inactive-text="Import (Do not ignore)" :disabled="!editingRule.modifyIgnore" />
          </div>
        </el-form-item>
        <div v-if="!editingRule.modifyIgnore || !editingRule.ignore">
          <el-form-item label="Swap Accounts">
            <div style="display: flex; gap: 10px; width: 100%; align-items: center">
              <el-checkbox v-model="editingRule.modifySwapAccounts" />
              <el-switch v-model="editingRule.swapAccounts" active-text="Swap Asset and Opposing Accounts" :disabled="!editingRule.modifySwapAccounts" />
            </div>
          </el-form-item>
          <el-form-item label="Target Type">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="editingRule.modifyTargetType" />
              <el-select v-model="editingRule.targetType" placeholder="Select Type" clearable :disabled="!editingRule.modifyTargetType" style="flex: 1">
                <el-option label="withdrawal" value="withdrawal" />
                <el-option label="deposit" value="deposit" />
                <el-option label="transfer" value="transfer" />
              </el-select>
            </div>
          </el-form-item>
          <el-form-item label="Target Category">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="editingRule.modifyTargetCategory" />
              <el-select v-model="editingRule.targetCategory" filterable allow-create default-first-option placeholder="Select or type new Category" @change="updateRuleTargetCategory" :disabled="!editingRule.modifyTargetCategory" style="flex: 1">
                <el-option label="-- 清空 (Leave Empty) --" value="" style="color: #999; font-style: italic" />
                <el-option v-for="cat in ffCategories" :key="cat.id" :label="cat.name" :value="cat.name" />
              </el-select>
            </div>
          </el-form-item>
          <el-form-item label="Target Asset Acc">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="editingRule.modifyTargetAsset" />
              <el-select v-model="editingRule.targetAsset" filterable allow-create default-first-option placeholder="Select or type new Asset Account" @change="updateRuleTargetAsset" :disabled="!editingRule.modifyTargetAsset" style="flex: 1">
                <el-option label="-- 清空 (Leave Empty) --" value="" style="color: #999; font-style: italic" />
                <el-option-group label="🏦 Asset (资产)">
                  <el-option v-for="acc in assetAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
                </el-option-group>
                <el-option-group label="🛒 Expense (支出)">
                  <el-option v-for="acc in expenseAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
                </el-option-group>
                <el-option-group label="💰 Revenue (收入)">
                  <el-option v-for="acc in revenueAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
                </el-option-group>
              </el-select>
            </div>
          </el-form-item>
          <el-form-item label="Target Opposing Acc">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="editingRule.modifyTargetOpposing" />
              <el-select v-model="editingRule.targetOpposing" filterable allow-create default-first-option placeholder="Select or type new Opposing Account" @change="updateRuleTargetOpposing" :disabled="!editingRule.modifyTargetOpposing" style="flex: 1">
                <el-option label="-- 清空 (Leave Empty) --" value="" style="color: #999; font-style: italic" />
                <el-option-group label="🏦 Asset (资产)">
                  <el-option v-for="acc in assetAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
                </el-option-group>
                <el-option-group label="🛒 Expense (支出)">
                  <el-option v-for="acc in expenseAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
                </el-option-group>
                <el-option-group label="💰 Revenue (收入)">
                  <el-option v-for="acc in revenueAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
                </el-option-group>
              </el-select>
            </div>
          </el-form-item>
          <el-form-item label="Target Description">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="editingRule.modifyTargetDescription" />
              <el-input v-model="editingRule.targetDescription" placeholder="e.g. [${收/付款方式}] ${商品说明}" :disabled="!editingRule.modifyTargetDescription" style="flex: 1" />
            </div>
          </el-form-item>
          <el-form-item label="Target Notes">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="editingRule.modifyTargetNotes" />
              <el-input v-model="editingRule.targetNotes" placeholder="e.g. ${备注}" :disabled="!editingRule.modifyTargetNotes" style="flex: 1" />
            </div>
          </el-form-item>
          <el-form-item label="Add Tags">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="editingRule.modifyTargetTags" />
              <el-input v-model="editingRule.targetTags" placeholder="Comma separated tags" :disabled="!editingRule.modifyTargetTags" style="flex: 1" />
            </div>
          </el-form-item>
          </div>
        </template>

        <el-form-item label="Save To Group" v-if="!editingRule.isRuleGroup">
          <el-select v-model="saveToGroupName" placeholder="Root (Top Level)" clearable style="width: 100%">
            <el-option label="Root (Top Level)" value="" />
            <el-option v-for="g in allGroups" :key="g.name" :label="g.label" :value="g.name" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="ruleDialogVisible = false">Cancel</el-button>
          <el-button type="primary" @click="saveEditingRule">Confirm</el-button>
        </span>
      </template>
    </el-dialog>
    <!-- Transaction Editor Dialog -->
    <!-- Bulk Edit Dialog -->
    <el-dialog v-model="bulkEditDialogVisible" title="Bulk Edit Transactions" width="600px">
      <el-alert type="warning" :closable="false" style="margin-bottom: 20px;">
        You are editing <strong>{{ selectedTransactions.length }}</strong> transactions.
      </el-alert>
      <el-form :model="bulkEditModel" label-width="140px">
        <el-form-item label="Set Ignore Flag">
          <div style="display: flex; gap: 10px; width: 100%; align-items: center">
            <el-checkbox v-model="bulkEditModel.modifyIgnore" />
            <el-switch v-model="bulkEditModel.ignore" active-text="Skip (Ignore)" inactive-text="Import (Do not ignore)" :disabled="!bulkEditModel.modifyIgnore" />
          </div>
        </el-form-item>
        <div v-if="!bulkEditModel.modifyIgnore || !bulkEditModel.ignore">
          <el-form-item label="Swap Accounts">
            <div style="display: flex; gap: 10px; width: 100%; align-items: center">
              <el-checkbox v-model="bulkEditModel.modifySwapAccounts" />
              <el-switch v-model="bulkEditModel.swapAccounts" active-text="Swap Asset and Opposing Accounts" :disabled="!bulkEditModel.modifySwapAccounts" />
            </div>
          </el-form-item>
          <el-form-item label="Target Type">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="bulkEditModel.modifyType" />
              <el-select v-model="bulkEditModel.type" placeholder="Select Type" clearable :disabled="!bulkEditModel.modifyType" style="flex: 1">
                <el-option label="withdrawal" value="withdrawal" />
                <el-option label="deposit" value="deposit" />
                <el-option label="transfer" value="transfer" />
              </el-select>
            </div>
          </el-form-item>
          <el-form-item label="Target Category">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="bulkEditModel.modifyCategory" />
              <el-select v-model="bulkEditModel.category" filterable allow-create default-first-option placeholder="Select or type new Category" @change="(val: string) => { bulkEditModel.categoryId = findCatId(val) }" :disabled="!bulkEditModel.modifyCategory" style="flex: 1">
                <el-option label="-- 清空 (Leave Empty) --" value="" style="color: #999; font-style: italic" />
                <el-option v-for="cat in ffCategories" :key="cat.id" :label="cat.name" :value="cat.name" />
              </el-select>
            </div>
          </el-form-item>
          <el-form-item label="Target Asset Acc">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="bulkEditModel.modifyAsset" />
              <el-select v-model="bulkEditModel.assetAccount" filterable allow-create default-first-option placeholder="Select or type new Asset Account" @change="(val: string) => { bulkEditModel.assetId = findAccId(val) }" :disabled="!bulkEditModel.modifyAsset" style="flex: 1">
                <el-option label="-- 清空 (Leave Empty) --" value="" style="color: #999; font-style: italic" />
                <el-option v-for="acc in ffAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
              </el-select>
            </div>
          </el-form-item>
          <el-form-item label="Target Opposing Acc">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="bulkEditModel.modifyOpposing" />
              <el-select v-model="bulkEditModel.opposingAccount" filterable allow-create default-first-option placeholder="Select or type new Opposing Account" @change="(val: string) => { bulkEditModel.opposingId = findAccId(val) }" :disabled="!bulkEditModel.modifyOpposing" style="flex: 1">
                <el-option label="-- 清空 (Leave Empty) --" value="" style="color: #999; font-style: italic" />
                <el-option v-for="acc in ffAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
              </el-select>
            </div>
          </el-form-item>
          <el-form-item label="Target Tags">
            <div style="display: flex; gap: 10px; width: 100%">
              <el-checkbox v-model="bulkEditModel.modifyTags" />
              <el-input v-model="bulkEditModel.tags" placeholder="Comma separated tags" :disabled="!bulkEditModel.modifyTags" style="flex: 1" />
            </div>
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="bulkEditDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="applyBulkEdit">Apply to Selected</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="txDialogVisible" title="Edit Transaction" width="70%">
      <div style="display: flex; gap: 20px;">
        <div style="flex: 1">
          <h4>Original Data</h4>
          <el-descriptions :column="1" border size="small" direction="horizontal">
            <el-descriptions-item v-for="(value, key) in parseRawData(editingTx.rawData)" :key="key" :label="key" label-width="120px" label-align="right">
              {{ value }}
            </el-descriptions-item>
          </el-descriptions>
        </div>
        <div style="flex: 2">
          <el-form :model="editingTx" label-width="140px">
            <el-form-item label="Date">
              <el-date-picker v-model="editingTx.date" type="datetime" />
            </el-form-item>
            <el-form-item label="Type">
          <el-radio-group v-model="editingTx.type">
            <el-radio value="withdrawal">Withdrawal</el-radio>
            <el-radio value="deposit">Deposit</el-radio>
            <el-radio value="transfer">Transfer</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="Amount">
          <el-input v-model="editingTx.amount" />
        </el-form-item>
        <el-form-item label="Category">
          <el-select v-model="editingTx.category" filterable allow-create>
            <el-option label="-- 未确定 (Leave Empty) --" value="" style="color: #999; font-style: italic" />
            <el-option v-for="cat in ffCategories" :key="cat.id" :label="cat.name" :value="cat.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="Asset Account">
          <el-select v-model="editingTx.assetAccount" filterable allow-create>
            <el-option label="-- 未确定 (Leave Empty) --" value="" style="color: #999; font-style: italic" />
            <el-option v-for="acc in assetAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="Opposing Account">
          <el-select v-model="editingTx.opposingAccount" filterable allow-create>
            <el-option label="-- 未确定 (Leave Empty) --" value="" style="color: #999; font-style: italic" />
            <el-option-group label="Expense (支出)" v-if="editingTx.type === 'withdrawal'">
              <el-option v-for="acc in expenseAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
            </el-option-group>
            <el-option-group label="Revenue (收入)" v-if="editingTx.type === 'deposit'">
              <el-option v-for="acc in revenueAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
            </el-option-group>
            <el-option-group label="Asset/Liability (转账)">
              <el-option v-for="acc in assetAccounts" :key="acc.id" :label="acc.name" :value="acc.name" />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="Description">
          <el-input v-model="editingTx.description" />
        </el-form-item>
        <el-form-item label="Notes">
          <el-input v-model="editingTx.notes" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      </div>
      </div>
      <template #footer>
        <el-button @click="txDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="saveTxEditor">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, provide, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, InfoFilled } from '@element-plus/icons-vue'
import axios from 'axios'
import RuleGroup from './components/RuleGroup.vue'

import RuleNode from './components/RuleNode.vue'

const API_BASE = 'http://localhost:8080/api'

const activeTab = ref('import')
const config = ref({
  fireflyUrl: '',
  fireflyToken: '',
  globalTags: '',
  deduplicateByExternalId: false,
  alipayExternalIdField: '',
  wechatExternalIdField: '',
  defaultAssetAccount: '',
  defaultAssetAccountId: '',
  defaultOpposingAccount: '',
  defaultOpposingId: '',
  mappingRules: [] as any[]
})
const savingConfig = ref(false)

const transactions = ref<any[]>([])
const originalTransactions = ref<any[]>([])
const selectedTransactions = ref<any[]>([])
const importing = ref(false)
const currentFileIsAlipay = ref(false)

const searchQuery = ref('')
const filteredTransactions = computed(() => {
  if (!searchQuery.value) return transactions.value
  const query = searchQuery.value.toLowerCase()
  return transactions.value.filter(tx => {
    if (tx.rawData && tx.rawData.toLowerCase().includes(query)) return true
    if (tx.description && tx.description.toLowerCase().includes(query)) return true
    if (tx.notes && tx.notes.toLowerCase().includes(query)) return true
    if (tx.counterparty && tx.counterparty.toLowerCase().includes(query)) return true
    if (tx.category && tx.category.toLowerCase().includes(query)) return true
    if (tx.assetAccount && tx.assetAccount.toLowerCase().includes(query)) return true
    if (tx.opposingAccount && tx.opposingAccount.toLowerCase().includes(query)) return true
    if (tx.amount && tx.amount.toLowerCase().includes(query)) return true
    return false
  })
})

const bulkEditDialogVisible = ref(false)
const bulkEditModel = ref({
  modifyType: false, type: '',
  modifyCategory: false, category: '', categoryId: '',
  modifyAsset: false, assetAccount: '', assetId: '',
  modifyOpposing: false, opposingAccount: '', opposingId: '',
  modifyTags: false, tags: '',
  modifyIgnore: false, ignore: false,
  modifySwapAccounts: false, swapAccounts: false
})

const openBulkEditor = () => {
  bulkEditModel.value = {
    modifyType: false, type: '',
    modifyCategory: false, category: '', categoryId: '',
    modifyAsset: false, assetAccount: '', assetId: '',
    modifyOpposing: false, opposingAccount: '', opposingId: '',
    modifyTags: false, tags: '',
    modifyIgnore: false, ignore: false,
    modifySwapAccounts: false, swapAccounts: false
  }
  bulkEditDialogVisible.value = true
}

const applyBulkEdit = () => {
  const model = bulkEditModel.value
  selectedTransactions.value.forEach(tx => {
    // Modify proxy directly
    if (model.modifyType && model.type) {
      tx.type = model.type
    }
    if (model.modifyCategory) {
      tx.category = model.category
      tx.categoryId = model.categoryId
    }
    if (model.modifyAsset) {
      tx.assetAccount = model.assetAccount
      tx.assetId = model.assetId
    }
    if (model.modifyOpposing) {
      tx.opposingAccount = model.opposingAccount
      tx.opposingId = model.opposingId
    }
    if (model.modifyTags) {
      if (model.tags.trim() === '') {
        tx.tags = []
      } else {
        tx.tags = model.tags.split(',').map(t => t.trim()).filter(t => t !== '')
      }
    }
    if (model.modifyIgnore) {
      tx.ignore = model.ignore
    }
    if (model.modifySwapAccounts && model.swapAccounts) {
      const tempAcc = tx.assetAccount
      const tempId = tx.assetId
      tx.assetAccount = tx.opposingAccount
      tx.assetId = tx.opposingId
      tx.opposingAccount = tempAcc
      tx.opposingId = tempId
    }
  })
  
  bulkEditDialogVisible.value = false
  ElMessage.success(`Successfully updated ${selectedTransactions.value.length} transactions.`)
}

const txDialogVisible = ref(false)
const editingTxIndex = ref(-1)
const editingTx = ref<any>({})

const ffCategories = ref<{id: string, name: string}[]>([])
const ffAccounts = ref<{id: string, name: string, type: string}[]>([])
const syncing = ref(false)

const assetAccounts = computed(() => ffAccounts.value.filter(a => a.type === 'asset'))
const expenseAccounts = computed(() => ffAccounts.value.filter(a => a.type === 'expense'))
const revenueAccounts = computed(() => ffAccounts.value.filter(a => a.type === 'revenue'))

const validTransactionsCount = computed(() => {
  return transactions.value.filter(t => !t.ignore).length
})

const ruleDialogVisible = ref(false)
const editingRule = ref<any>({})
const editingRuleIndex = ref(-1)

onMounted(async () => {
  await loadConfig()
  if (config.value.fireflyUrl && config.value.fireflyToken) {
    await syncFireflyData()
  }
})

const loadConfig = async () => {
  try {
    const res = await axios.get(`${API_BASE}/config`)
    if (res.data) {
      config.value = {
        fireflyUrl: res.data.fireflyUrl || '',
        fireflyToken: res.data.fireflyToken || '',
        globalTags: res.data.globalTags || '',
        deduplicateByExternalId: res.data.deduplicateByExternalId || false,
        alipayExternalIdField: res.data.alipayExternalIdField || '',
        wechatExternalIdField: res.data.wechatExternalIdField || '',
        defaultAssetAccount: res.data.defaultAssetAccount || res.data.defaultAlipayAccount || '',
        defaultAssetAccountId: res.data.defaultAssetAccountId || res.data.defaultAlipayAccountId || '',
        defaultOpposingAccount: res.data.defaultOpposingAccount || '',
        defaultOpposingId: res.data.defaultOpposingId || '',
        mappingRules: res.data.mappingRules || []
      }
    }
  } catch (error) {
    console.error('Failed to load config', error)
  }
}

const syncFireflyData = async () => {
  if (!config.value.fireflyUrl || !config.value.fireflyToken) {
    ElMessage.warning('Please save your Firefly URL and Token first.')
    return
  }
  syncing.value = true
  try {
    const [catRes, accRes] = await Promise.all([
      axios.get(`${API_BASE}/firefly/categories`),
      axios.get(`${API_BASE}/firefly/accounts`)
    ])
    ffCategories.value = catRes.data || []
    ffAccounts.value = accRes.data || []
    ElMessage.success('FireFly III data synced successfully!')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || 'Failed to sync FireFly III data. Check config.')
  } finally {
    syncing.value = false
  }
}

// Helpers to map name back to ID
const findCatId = (name: string) => ffCategories.value.find(c => c.name === name)?.id || ''
const findAccId = (name: string) => ffAccounts.value.find(a => a.name === name)?.id || ''

const updateConfigAssetAcc = (val: string) => { config.value.defaultAssetAccountId = findAccId(val) }
const updateConfigOpposingAcc = (val: string) => { config.value.defaultOpposingId = findAccId(val) }

const openTxEditor = (row: any) => {
  editingTxIndex.value = transactions.value.indexOf(row)
  editingTx.value = JSON.parse(JSON.stringify(row))
  txDialogVisible.value = true
}

const saveTxEditor = () => {
  editingTx.value.categoryId = findCatId(editingTx.value.category)
  editingTx.value.assetId = findAccId(editingTx.value.assetAccount)
  editingTx.value.opposingId = findAccId(editingTx.value.opposingAccount)
  
  if (editingTx.value.date instanceof Date) {
    editingTx.value.date = editingTx.value.date.toISOString()
  }

  transactions.value[editingTxIndex.value] = editingTx.value
  txDialogVisible.value = false
}

const updateRuleTargetCategory = (val: string) => { editingRule.value.targetCategoryId = findCatId(val) }
const updateRuleTargetAsset = (val: string) => { editingRule.value.targetAssetId = findAccId(val) }
const updateRuleTargetOpposing = (val: string) => { editingRule.value.targetOpposingId = findAccId(val) }

const loadPresets = async () => {
  try {
    await axios.post(`${API_BASE}/config/presets`)
    ElMessage.success('Default presets loaded successfully')
    await loadConfig() // reload rules
    if (transactions.value.length > 0) {
      await evaluateTransactions()
    }
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || 'Failed to load presets')
  }
}

const exportConfig = () => {
  const dataStr = JSON.stringify(config.value, null, 2)
  const blob = new Blob([dataStr], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `firefly-importer-config-${new Date().toISOString().slice(0, 10)}.json`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('Config exported')
}

const importConfig = (file: File) => {
  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const imported = JSON.parse(e.target?.result as string)
      // Send to backend which will backup current config first
      await axios.post(`${API_BASE}/config/import`, imported)
      ElMessage.success('Config imported successfully. Previous config backed up.')
      await loadConfig()
      if (transactions.value.length > 0) {
        await evaluateTransactions()
      }
    } catch (error: any) {
      ElMessage.error(error.response?.data?.error || 'Failed to import config')
    }
  }
  reader.readAsText(file)
  return false // prevent el-upload auto upload
}

const saveConfig = async () => {
  savingConfig.value = true
  try {
    await axios.post(`${API_BASE}/config`, config.value)
    ElMessage.success('Configuration saved successfully')
    if (transactions.value.length > 0) {
      await evaluateTransactions()
    }
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || 'Failed to save config')
  } finally {
    savingConfig.value = false
  }
}




const dragState = reactive({
  sourcePath: null as number[] | null
})
provide('dragState', dragState)

const handleTreeDrop = (sourcePath: number[], targetPath: number[], zone: string) => {
  console.log('[DnD] drop', { sourcePath: [...sourcePath], targetPath: [...targetPath], zone })

  // Prevent dropping onto self or into own children
  if (JSON.stringify(sourcePath) === JSON.stringify(targetPath)) return
  const sourceStr = JSON.stringify(sourcePath)
  const targetStr = JSON.stringify(targetPath)
  if (targetStr.startsWith(sourceStr.slice(0, -1) + ',') || targetStr === sourceStr) return

  const getArray = (path: number[]): any[] | null => {
    let arr: any[] = config.value.mappingRules
    for (let i = 0; i < path.length - 1; i++) {
      const node = arr[path[i]]
      if (!node) { console.warn('[DnD] node missing at', path[i], 'in path', path); return null }
      if (!node.rules) node.rules = []
      arr = node.rules
    }
    return arr
  }

  // 1. Resolve source
  const sourceArr = getArray(sourcePath)
  if (!sourceArr) { console.warn('[DnD] sourceArr null'); return }
  const sourceIdx = sourcePath[sourcePath.length - 1]
  if (sourceIdx < 0 || sourceIdx >= sourceArr.length) { console.warn('[DnD] sourceIdx OOB', sourceIdx, sourceArr.length); return }
  const itemToMove = sourceArr[sourceIdx]

  // 2. For 'right' (drop INTO group), grab direct reference BEFORE any mutation
  let targetGroupRef: any = null
  if (zone === 'right') {
    const tArr = getArray(targetPath)
    if (!tArr) { console.warn('[DnD] target arr null for right'); return }
    const tIdx = targetPath[targetPath.length - 1]
    if (tIdx < 0 || tIdx >= tArr.length) { console.warn('[DnD] tIdx OOB', tIdx, tArr.length); return }
    targetGroupRef = tArr[tIdx]
  }

  // 3. Remove from source
  sourceArr.splice(sourceIdx, 1)

  // 4. Adjust target path to account for index shift from removal
  //    If source was in the same parent array as an ancestor of target,
  //    and source index < target's index at that level, decrement it.
  const adjustedTarget = [...targetPath]
  const srcDepth = sourcePath.length - 1
  let sameParent = true
  for (let i = 0; i < srcDepth; i++) {
    if (i >= adjustedTarget.length || sourcePath[i] !== adjustedTarget[i]) {
      sameParent = false
      break
    }
  }
  if (sameParent && srcDepth < adjustedTarget.length && sourceIdx < adjustedTarget[srcDepth]) {
    adjustedTarget[srcDepth]--
    console.log('[DnD] adjusted target path from', targetPath, 'to', adjustedTarget)
  }

  // 5. Insert
  if (zone === 'right' && targetGroupRef) {
    if (!targetGroupRef.rules) targetGroupRef.rules = []
    targetGroupRef.rules.push(itemToMove)
    console.log('[DnD] pushed into group', targetGroupRef.name)
  } else {
    const targetArr = getArray(adjustedTarget)
    if (!targetArr) { console.warn('[DnD] targetArr null after splice'); return }
    let tIdx = adjustedTarget[adjustedTarget.length - 1]
    if (tIdx > targetArr.length) tIdx = targetArr.length
    targetArr.splice(tIdx, 0, itemToMove)
    console.log('[DnD] inserted at index', tIdx)
  }

  saveConfig()
  console.log('[DnD] saveConfig called')
}
provide('handleTreeDrop', handleTreeDrop)

const editingRuleParent = ref<any>(null)
const saveToGroupName = ref('')

const collectGroups = (rules: any[], depth: number = 0, result: any[] = []): any[] => {
  for (const r of rules) {
    if (r.isRuleGroup) {
      const indent = '\u3000'.repeat(depth) + (depth > 0 ? '\u2514 ' : '')
      result.push({ name: r.name, label: indent + (r.name || 'Untitled Group'), ref: r })
      if (r.rules) collectGroups(r.rules, depth + 1, result)
    }
  }
  return result
}
const allGroups = computed(() => collectGroups(config.value.mappingRules || []))

const openRuleEditor = (rule: any, parent: any, index: number) => {
  saveToGroupName.value = parent ? (parent.name || '') : ''
  editingRuleParent.value = parent
  editingRuleIndex.value = index
  if (rule) {
    editingRule.value = JSON.parse(JSON.stringify(rule))
    if (!editingRule.value.conditions) editingRule.value.conditions = []
    if (!editingRule.value.conditionLogic) editingRule.value.conditionLogic = 'AND'
    
    // Set UI toggles based on existing data
    editingRule.value.modifyIgnore = 'modifyIgnore' in rule ? rule.modifyIgnore : !!rule.ignore
    editingRule.value.modifySwapAccounts = 'modifySwapAccounts' in rule ? rule.modifySwapAccounts : !!rule.swapAccounts
    editingRule.value.modifyTargetType = !!rule.targetType
    editingRule.value.modifyTargetCategory = !!rule.targetCategory || !!rule.targetCategoryId
    editingRule.value.modifyTargetAsset = !!rule.targetAsset || !!rule.targetAssetId
    editingRule.value.modifyTargetOpposing = !!rule.targetOpposing || !!rule.targetOpposingId
    editingRule.value.modifyTargetDescription = !!rule.targetDescription
    editingRule.value.modifyTargetNotes = !!rule.targetNotes
    editingRule.value.modifyTargetTags = !!rule.targetTags
  } else {
    editingRule.value = {
      name: '',
      isRuleGroup: false,
      excludeAlipay: false, excludeWechat: false,
      conditionLogic: 'AND',
      conditions: [
        { matchField: '', matchType: 'Contains', matchValue: '' }
      ],
      ignore: false, modifyIgnore: false, 
      swapAccounts: false, modifySwapAccounts: false,
      targetType: '', modifyTargetType: false,
      targetCategory: '', targetCategoryId: '', modifyTargetCategory: false,
      targetAsset: '', targetAssetId: '', modifyTargetAsset: false,
      targetOpposing: '', targetOpposingId: '', modifyTargetOpposing: false,
      targetDescription: '', modifyTargetDescription: false,
      targetNotes: '', modifyTargetNotes: false,
      targetTags: '', modifyTargetTags: false
    }
  }
  ruleDialogVisible.value = true
}

const addGroupTo = (parent: any) => {
  openRuleEditor(null, parent, -1)
  editingRule.value.isRuleGroup = true
}

const handleEditChild = ({ rule, parent, index }: any) => {
  openRuleEditor(rule, parent, index)
}

const removeRootRule = (idx: number) => {
  config.value.mappingRules.splice(idx, 1)
  saveConfig()
}

const moveRootRuleUp = (idx: number) => {
  if (idx > 0) {
    const temp = config.value.mappingRules[idx]
    config.value.mappingRules[idx] = config.value.mappingRules[idx - 1]
    config.value.mappingRules[idx - 1] = temp
    saveConfig()
  }
}

const moveRootRuleDown = (idx: number) => {
  if (idx < config.value.mappingRules.length - 1) {
    const temp = config.value.mappingRules[idx]
    config.value.mappingRules[idx] = config.value.mappingRules[idx + 1]
    config.value.mappingRules[idx + 1] = temp
    saveConfig()
  }
}



const saveEditingRule = async () => {
  const ruleToSave = { ...editingRule.value }

  if (!ruleToSave.modifyTargetType) { ruleToSave.targetType = '' }
  if (!ruleToSave.modifyTargetCategory) { ruleToSave.targetCategory = ''; ruleToSave.targetCategoryId = '' }
  if (!ruleToSave.modifyTargetAsset) { ruleToSave.targetAsset = ''; ruleToSave.targetAssetId = '' }
  if (!ruleToSave.modifyTargetOpposing) { ruleToSave.targetOpposing = ''; ruleToSave.targetOpposingId = '' }
  if (!ruleToSave.modifyTargetDescription) { ruleToSave.targetDescription = '' }
  if (!ruleToSave.modifyTargetNotes) { ruleToSave.targetNotes = '' }
  if (!ruleToSave.modifyTargetTags) { ruleToSave.targetTags = '' }

  delete ruleToSave.modifyTargetType
  delete ruleToSave.modifyTargetCategory
  delete ruleToSave.modifyTargetAsset
  delete ruleToSave.modifyTargetOpposing
  delete ruleToSave.modifyTargetDescription
  delete ruleToSave.modifyTargetNotes
  delete ruleToSave.modifyTargetTags

  // Determine where to save
  const currentParentName = editingRuleParent.value ? (editingRuleParent.value.name || '') : ''
  const newParentName = saveToGroupName.value || ''

  // If editing an existing rule AND the group changed, remove from old location first
  if (editingRuleIndex.value >= 0 && currentParentName !== newParentName) {
    let oldArray = config.value.mappingRules as any[]
    if (editingRuleParent.value) {
      oldArray = editingRuleParent.value.rules || []
    }
    oldArray.splice(editingRuleIndex.value, 1)
    // Now treat as a new rule (append to target)
    editingRuleIndex.value = -1
  }

  let targetArray = config.value.mappingRules as any[]
  if (newParentName) {
    const group = allGroups.value.find((g: any) => g.name === newParentName)
    if (group) {
      if (!group.ref.rules) group.ref.rules = []
      targetArray = group.ref.rules
    }
  } else if (editingRuleIndex.value >= 0 && editingRuleParent.value) {
    if (!editingRuleParent.value.rules) editingRuleParent.value.rules = []
    targetArray = editingRuleParent.value.rules
  }

  if (editingRuleIndex.value >= 0) {
    targetArray[editingRuleIndex.value] = ruleToSave
  } else {
    targetArray.push(ruleToSave)
  }
  ruleDialogVisible.value = false
  await saveConfig()
}

const createRuleFromRow = (row: any) => {
  editingRuleIndex.value = -1
  editingRuleParent.value = null
  saveToGroupName.value = ''
  
  const initialConditions: any[] = []
  if (row.rawData) {
    try {
      const rawMap = JSON.parse(row.rawData)
      for (const [key, value] of Object.entries(rawMap)) {
        if (value && typeof value === 'string' && value.trim() !== '') {
          initialConditions.push({ matchField: key, matchType: 'Equals', matchValue: value.trim() })
        }
      }
    } catch (e) {
      console.error('Failed to parse raw data for rule generation', e)
    }
  }

  if (initialConditions.length === 0) {
    initialConditions.push({ matchField: '', matchType: 'Equals', matchValue: '' })
  }

  editingRule.value = {
    name: `Rule for ${row.counterparty || 'Unknown'}`,
    conditionLogic: 'AND',
    conditions: initialConditions,
    ignore: false, modifyIgnore: false, 
    swapAccounts: false, modifySwapAccounts: false,
    targetType: '', modifyTargetType: false,
    targetCategory: row.category || '', targetCategoryId: row.categoryId || '', modifyTargetCategory: !!row.category,
    targetAsset: row.assetAccount || '', targetAssetId: row.assetId || '', modifyTargetAsset: !!row.assetAccount,
    targetOpposing: row.opposingAccount || '', targetOpposingId: row.opposingId || '', modifyTargetOpposing: !!row.opposingAccount,
    targetDescription: '', modifyTargetDescription: false,
    targetNotes: '', modifyTargetNotes: false,
    targetTags: '', modifyTargetTags: false
  }
  ruleDialogVisible.value = true
}



const handleUploadSuccess = (response: any, uploadFile: any) => {
  ElMessage.success('File uploaded and parsed successfully')
  
  const validCategoryNames = new Set(ffCategories.value.map((c: any) => c.name))
  const txs = response.transactions || []
  txs.forEach((tx: any) => {
    if (tx.category && !validCategoryNames.has(tx.category)) {
      tx.category = ''
      tx.categoryId = ''
    }
  })

  transactions.value = txs
  originalTransactions.value = JSON.parse(JSON.stringify(txs))
  currentFileIsAlipay.value = uploadFile.name.toLowerCase().endsWith('.csv')
}

const evaluateTransactions = async () => {
  try {
    const res = await axios.post(`${API_BASE}/evaluate`, {
      transactions: transactions.value,
      isAlipay: currentFileIsAlipay.value
    })
    transactions.value = res.data.transactions
    ElMessage.success('Preview updated with new rules')
  } catch (error) {
    console.error('Failed to evaluate rules', error)
  }
}

const resetAndEvaluateTransactions = async () => {
  if (originalTransactions.value.length === 0) return
  try {
    const res = await axios.post(`${API_BASE}/evaluate`, {
      transactions: JSON.parse(JSON.stringify(originalTransactions.value)),
      isAlipay: currentFileIsAlipay.value
    })
    transactions.value = res.data.transactions
    ElMessage.success('Transactions reset and rules reapplied')
  } catch (error) {
    console.error('Failed to reset and evaluate rules', error)
  }
}

const txTable = ref<any>(null)

const handleSelectionChange = (val: any[]) => {
  selectedTransactions.value = val
}

const handleCellClick = (row: any, column: any) => {
  // Don't toggle for interactive columns (selection checkbox, actions)
  const skipLabels = ['', 'Actions']
  if (skipLabels.includes(column.label)) return
  if (txTable.value) {
    txTable.value.toggleRowSelection(row)
  }
}

const resetSelectedTransactions = async () => {
  if (selectedTransactions.value.length === 0) return
  
  const selectedOriginals = selectedTransactions.value.map(tx => {
    const idx = transactions.value.indexOf(tx)
    return JSON.parse(JSON.stringify(originalTransactions.value[idx]))
  })

  try {
    const res = await axios.post(`${API_BASE}/evaluate`, {
      transactions: selectedOriginals,
      isAlipay: currentFileIsAlipay.value
    })
    
    selectedTransactions.value.forEach((tx, i) => {
      const originalIdx = transactions.value.indexOf(tx)
      transactions.value[originalIdx] = res.data.transactions[i]
    })
    ElMessage.success('Selected transactions reset')
  } catch (error) {
    console.error('Failed to reset selected', error)
  }
}

const resetRow = async (row: any) => {
  const index = transactions.value.indexOf(row)
  const original = JSON.parse(JSON.stringify(originalTransactions.value[index]))
  try {
    const res = await axios.post(`${API_BASE}/evaluate`, {
      transactions: [original],
      isAlipay: currentFileIsAlipay.value
    })
    transactions.value[index] = res.data.transactions[0]
    ElMessage.success('Row reset')
  } catch (error) {
    console.error('Failed to reset row', error)
  }
}

const handleUploadError = (error: any) => {
  let msg = 'Upload failed'
  if (error.message) {
    try {
      const parsed = JSON.parse(error.message)
      if (parsed.error) msg = parsed.error
    } catch (e) {
      // ignore
    }
  }
  ElMessage.error(msg)
}

const importProgressVisible = ref(false)
const importProgress = reactive({
  current: 0,
  total: 0,
  success: 0,
  failed: 0,
  skipped: 0,
  percentage: 0,
  status: '' as '' | 'success' | 'exception' | 'warning'
})
const importResults = ref<any[]>([])
let importAbortController: AbortController | null = null

const stopImport = () => {
  if (importAbortController) {
    importAbortController.abort()
    importAbortController = null
  }
}

const submitImport = async (onlySelected: boolean = false) => {
  const targetTransactions = onlySelected ? selectedTransactions.value : transactions.value;
  
  if (targetTransactions.length === 0) return

  // Reset progress
  importProgress.current = 0
  importProgress.total = 0
  importProgress.success = 0
  importProgress.failed = 0
  importProgress.skipped = 0
  importProgress.percentage = 0
  importProgress.status = ''
  importResults.value = []
  importProgressVisible.value = true
  importing.value = true
  importAbortController = new AbortController()

  try {
    // Save config first to ensure dedup setting is applied
    await saveConfig()

    const response = await fetch(`${API_BASE}/import`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ transactions: targetTransactions }),
      signal: importAbortController.signal
    })

    if (!response.ok && !response.body) {
      ElMessage.error('Import request failed')
      importing.value = false
      return
    }

    const reader = response.body!.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          try {
            const data = JSON.parse(line.slice(6))
            if (data.index !== undefined) {
              importProgress.current = data.index + 1
              importProgress.total = data.total
              importProgress.percentage = Math.round(((data.index + 1) / data.total) * 100)

              if (data.status === 'success') importProgress.success++
              else if (data.status === 'error') importProgress.failed++
              else importProgress.skipped++

              importResults.value.push({
                description: data.description || data.txId,
                status: data.status,
                message: data.message
              })
            }
          } catch (e) {
            // skip parse errors
          }
        } else if (line.startsWith('event: done')) {
          // done event handled in next data line
        }
      }
    }

    // Final status
    if (importProgress.failed > 0) {
      importProgress.status = 'exception'
      ElMessage.warning(`Import completed. ${importProgress.success} succeeded, ${importProgress.failed} failed, ${importProgress.skipped} skipped.`)
    } else {
      importProgress.status = 'success'
      ElMessage.success(`Import completed! ${importProgress.success} transactions imported successfully.`)
    }
  } catch (error: any) {
    if (error.name === 'AbortError') {
      importProgress.status = 'warning'
      ElMessage.warning(`Import stopped. ${importProgress.success} already imported, ${importProgress.current} of ${importProgress.total} processed.`)
    } else {
      ElMessage.error('Import failed: ' + (error.message || 'Unknown error'))
      importProgress.status = 'exception'
    }
  } finally {
    importing.value = false
    importAbortController = null
  }
}



const tableRowClassName = ({ row }: { row: any }) => {
  if (row.ignore) {
    return 'ignored-row'
  }
  return ''
}

const parseRawData = (rawStr: string) => {
  if (!rawStr) return {}
  try {
    return JSON.parse(rawStr)
  } catch (e) {
    return { error: 'Failed to parse raw data' }
  }
}

const getSuggestionsForField = (field: string) => {
  if (!field) return []
  const values = new Set<string>()
  transactions.value.forEach(tx => {
    let val = ''
    switch(field) {
      case 'Counterparty': val = tx.counterparty; break;
      case 'Description': val = tx.description; break;
      case 'Category': val = tx.category; break;
      case 'PaymentMethod': val = tx.paymentMethod; break;
      case 'Status': val = tx.status; break;
      case 'TransactionID': val = tx.transactionId; break;
      case 'Amount': val = tx.amount; break;
      case 'Notes': val = tx.notes; break;
      case 'Any': return;
      default:
        try {
          if (tx.rawData) {
            const raw = JSON.parse(tx.rawData)
            if (raw[field]) val = raw[field]
          }
        } catch(e) {}
    }
    if (val) values.add(val)
  })
  return Array.from(values)
}

const querySuggestions = (field: string, queryString: string, cb: any) => {
  const suggestions = getSuggestionsForField(field).map(v => ({ value: v }))
  const results = queryString
    ? suggestions.filter(s => s.value.toLowerCase().includes(queryString.toLowerCase()))
    : suggestions
  cb(results)
}

const fetchSuggestions = (field: string) => {
  return (queryString: string, cb: any) => {
    querySuggestions(field, queryString, cb)
  }
}

const filterType = (value: string, row: any) => row.type === value
const filterCategory = (value: string, row: any) => value === 'empty' ? !row.category : !!row.category
const filterAccounts = (value: string, row: any) => {
  if (value === 'missing_asset') return !row.assetAccount
  if (value === 'missing_opposing') return !row.opposingAccount
  if (value === 'default_asset') return row.assetAccount === config.value.defaultAssetAccount && !!config.value.defaultAssetAccount
  if (value === 'default_opposing') return row.opposingAccount === config.value.defaultOpposingAccount && !!config.value.defaultOpposingAccount
  return true
}
const filterStatus = (value: boolean, row: any) => row.ignore === value

const rawFieldKeys = computed(() => {
  const keys = new Set<string>()
  transactions.value.forEach(tx => {
    if (tx.rawData) {
      try {
        const raw = JSON.parse(tx.rawData)
        Object.keys(raw).forEach(k => keys.add(k))
      } catch (e) {}
    }
  })
  return Array.from(keys)
})
</script>

<style>
.upload-section {
  display: flex;
  justify-content: center;
  padding: 2rem 0;
}
.actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 1rem;
}
.rules-list {
  display: flex;
  flex-wrap: wrap;
  gap: 15px;
}
.condition-box {
  margin-left: 140px;
  margin-bottom: 10px;
  background: #f8fafc;
  padding: 10px;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
}
.el-table .ignored-row {
  opacity: 0.5;
  background-color: #fafafa !important;
}
.el-table .ignored-row td {
  text-decoration: line-through;
}
h4 {
  margin-top: 25px;
  margin-bottom: 15px;
  border-bottom: 1px solid #eee;
  padding-bottom: 5px;
}
.raw-data-panel {
  padding: 10px 40px;
  background-color: #f8fafc;
}
.raw-data-panel h4 {
  margin-top: 0;
  margin-bottom: 10px;
  color: #475569;
}
</style>
