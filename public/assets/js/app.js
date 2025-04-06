// 备份系统前端JS
'use strict';

// DOM引用
let tasksPanel, recordsPanel, configsPanel, systemConfigPanel;
let navTasks, navRecords, navSystemConfig;
let tasksTableBody, recordsTableBody, filterTask, pagination, taskPagination;
let storageType, localStorageConfig, s3StorageConfig;
let currentPage = 1, currentPageSize = 10, currentTaskId = null;
let tasksList = [];
let configsList = [];
let btnLogout;
let taskForm;

// 初始化应用
document.addEventListener('DOMContentLoaded', function() {

    // 初始化DOM引用
    tasksPanel = document.getElementById('tasks-panel');
    recordsPanel = document.getElementById('records-panel');
    systemConfigPanel = document.getElementById('system-config-panel');
    navTasks = document.getElementById('nav-tasks');
    navRecords = document.getElementById('nav-records');
    navSystemConfig = document.getElementById('nav-system-config');
    tasksTableBody = document.getElementById('tasks-table-body');
    recordsTableBody = document.getElementById('records-table-body');
    filterTask = document.getElementById('filter-task');
    pagination = document.getElementById('pagination');
    taskPagination = document.getElementById('task-pagination');
    storageType = document.getElementById('storage-type');
    localStorageConfig = document.getElementById('local-storage-config');
    s3StorageConfig = document.getElementById('s3-storage-config');
    taskForm = document.getElementById('task-form');

    // 检查认证状态
    checkAuthStatus();

    // 初始化Cron表达式验证功能
    initializeCronValidation();

    // 绑定事件监听器
    bindEventListeners();

    // 清理按钮
    bindManualCleanupButton();

    // 根据URL显示对应面板
    showInitialPanel();
});

// 根据URL参数显示初始面板
function showInitialPanel() {
    // 获取URL中的panel参数
    const url = new URL(window.location.href);
    const panel = url.searchParams.get('panel') || 'tasks';
    const page = parseInt(url.searchParams.get('page')) || 1;
    const pageSize = parseInt(url.searchParams.get('pageSize')) || 10;
    const taskId = url.searchParams.get('taskId') || '';
    
    // 根据URL参数显示面板
    showPanel(panel);
}

// 导航到指定面板
function navigateTo(panel, params = {}) {
    // 构建URL
    const url = new URL(window.location.href);

    // 如果是系统配置页面，只保留panel参数
    if (panel === 'system-config') {
        url.search = ''
        url.searchParams.set('panel', panel);
    } else {
        // 其他页面保留所有参数
        url.searchParams.set('panel', panel);

        // 添加其他参数
        for (const key in params) {
            if (params[key]) {
                url.searchParams.set(key, params[key]);
            } else {
                url.searchParams.delete(key);
            }
        }
    }

    // 更新历史记录并跳转
    window.history.pushState({}, '', url.toString());

    // 显示对应面板
    showPanel(panel);

    // 收起移动端菜单
    collapseNavbarMenu();
}

// 检查登录状态
function checkAuthStatus() {
    // 如果存在token就认为已登录
    const token = localStorage.getItem('backupSystemAuth');

    // 如果没有存储token，则跳转到登录页
    if (!token) {
        window.location.href = '/login.html';
        return;
    }

    // 将API请求包装在try-catch块中，确保任何异常情况下都会跳转到登录页面
    try {
        fetch('/api/auth/check', {
            headers: {
                'Authorization': 'Bearer ' + token
            }
        })
            .then(response => {
                if (!response.ok || response.status === 401) {
                    // 未授权或请求失败，显示会话过期模态框
                    localStorage.removeItem('backupSystemAuth');
                    showSessionExpiredModal();
                    return null;
                }
                return response.json();
            })
            .then(result => {
                if (result && result.code === 200) {
                    // 已登录，继续初始化
                    initializeApp();
                } else if (result && result.code === 401) {
                    localStorage.removeItem('backupSystemAuth');
                    showSessionExpiredModal();
                } else if (result) {
                    showToast('验证登录状态失败: ' + (result.msg || '未知错误'), 'warning');
                }
            })
            .catch(error => {
                console.error('验证登录失败:', error);
                // 可能是网络问题，显示错误提示但不立即跳转
                showToast('验证登录失败，请检查网络连接', 'danger');
            });
    } catch (error) {
        console.error('验证登录过程中发生异常:', error);
        showToast('发生异常，请重新登录', 'danger');
        setTimeout(() => {
            localStorage.removeItem('backupSystemAuth');
            window.location.href = '/login.html';
        }, 2000);
    }
}

// 添加登出按钮
function addLogoutButton() {
    // 创建登出按钮
    btnLogout = document.createElement('button');
    btnLogout.className = 'btn btn-outline-light ms-2 d-flex align-items-center';

    // 使用内联SVG作为图标
    btnLogout.innerHTML = `
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="me-1" viewBox="0 0 16 16">
            <path fill-rule="evenodd" d="M10 12.5a.5.5 0 0 1-.5.5h-8a.5.5 0 0 1-.5-.5v-9a.5.5 0 0 1 .5-.5h8a.5.5 0 0 1 .5.5v2a.5.5 0 0 0 1 0v-2A1.5 1.5 0 0 0 9.5 2h-8A1.5 1.5 0 0 0 0 3.5v9A1.5 1.5 0 0 0 1.5 14h8a1.5 1.5 0 0 0 1.5-1.5v-2a.5.5 0 0 0-1 0v2z"/>
            <path fill-rule="evenodd" d="M15.854 8.354a.5.5 0 0 0 0-.708l-3-3a.5.5 0 0 0-.708.708L14.293 7.5H5.5a.5.5 0 0 0 0 1h8.793l-2.147 2.146a.5.5 0 0 0 .708.708l3-3z"/>
        </svg>
        退出登录
    `;

    // 添加登出事件
    btnLogout.addEventListener('click', function () {
        // 清除token
        localStorage.removeItem('backupSystemAuth');
        // 跳转到登录页
        window.location.href = '/login.html';
    });

    // 将按钮添加到导航栏
    const navbarNav = document.getElementById('navbarNav');
    if (navbarNav) {
        const navbarRight = document.createElement('div');
        navbarRight.className = 'ms-auto';
        navbarRight.appendChild(btnLogout);
        navbarNav.appendChild(navbarRight);
    }
}

// 检查用户是否已认证
function checkAuthentication() {
    return new Promise((resolve) => {
        // 如果存在token就认为已登录
        const token = localStorage.getItem('backupSystemAuth');

        // 如果没有token，跳转到登录页
        if (!token) {
            window.location.href = '/login.html';
            resolve(false);
            return;
        }

        // 添加登出按钮
        addLogoutButton();

        // 认证成功
        resolve(true);
    });
}

// 初始化应用
function initializeApp() {
    // 获取站点信息（在验证前获取，确保登录页也显示）
    getSiteInfo();
    
    // 检查是否需要认证
    checkAuthentication()
        .then(authenticated => {
            if (authenticated) {
                // 初始化Cron表达式验证功能
                initializeCronValidation();
                
                // 绑定事件监听器
                bindEventListeners();
                
                // 绑定手动清理按钮事件
                bindManualCleanupButton();
                
                // 显示初始面板
                showInitialPanel();
            }
        });
}

// 初始化Cron表达式验证
function initializeCronValidation() {
    // 简化版本：移除函数内容，保留函数定义以避免引用错误
}

// 绑定事件监听
function bindEventListeners() {
    // 导航事件
    navTasks.addEventListener('click', (e) => {
        e.preventDefault();
        navigateTo('tasks');
    });

    navRecords.addEventListener('click', (e) => {
        e.preventDefault();
        navigateTo('records');
    });

    navSystemConfig.addEventListener('click', (e) => {
        e.preventDefault();
        navigateTo('system-config');
    });

    // 根据存储类型切换配置面板
    document.getElementById('storage-type').addEventListener('change', toggleStorageConfigPanels);

    // Webhook 启用状态改变时控制表单字段
    document.getElementById('webhook-enabled').addEventListener('change', updateWebhookFormFields);

    // 测试 Webhook 事件
    document.getElementById('btn-test-webhook').addEventListener('click', testWebhook);

    // 系统配置保存按钮
    document.getElementById('btn-save-system-config').addEventListener('click', saveSystemConfig);

    // 添加任务按钮
    document.getElementById('btn-add-task').addEventListener('click', () => {
        resetTaskForm();
        document.getElementById('task-modal-title').textContent = '新建任务';
        showModal('task-modal');
    });

    // 任务保存按钮
    document.getElementById('btn-save-task').addEventListener('click', saveTask);

    // 根据任务类型切换配置面板
    document.getElementById('task-type').addEventListener('change', toggleConfigPanels);

    // Cron 表达式示例按钮
    document.querySelectorAll('.cron-example').forEach(button => {
        button.addEventListener('click', (e) => {
            e.preventDefault();
            document.getElementById('task-schedule').value = button.getAttribute('data-cron');

            // 验证 Cron 表达式
            validateCronInput();
        });
    });

    // Cron 表达式输入框验证
    document.getElementById('task-schedule').addEventListener('input', validateCronInput);

    // 任务记录筛选事件
    filterTask.addEventListener('change', () => {
        // 使用URL参数导航而不是使用localStorage
        const taskId = filterTask.value;
        navigateTo('records', {taskId: taskId, page: 1});
    });

    // 会话过期模态框中的重新登录按钮
    document.getElementById('btn-relogin').addEventListener('click', () => {
        window.location.reload();
    });
}

// 测试Webhook
function testWebhook() {
    const webhookUrl = document.getElementById('webhook-url').value;
    const webhookHeaders = document.getElementById('webhook-headers').value;
    const webhookBody = document.getElementById('webhook-body').value;

    if (!webhookUrl) {
        showToast('请填写Webhook URL', 'warning');
        return;
    }

    // 显示加载提示
    showLoading('正在测试Webhook...');

    // 构造当前表单配置
    const tempConfig = {
        url: webhookUrl,
        headers: webhookHeaders,
        body: webhookBody
    };

    // 发送请求
    apiRequest('/api/configs/testWebhook', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(tempConfig)
    })
        .then(result => {
            hideLoading();
            if (result && result.code === 200) {
                showToast('Webhook测试发送成功', 'success');
            } else {
                showToast(`测试失败: ${result?.msg || '未知错误'}`, 'danger');
            }
        })
        .catch(error => {
            hideLoading();
            showToast(`测试失败: ${error.message}`, 'danger');
        });
}

// 筛选任务记录

// 导航到指定面板
function navigateTo(panel, params = {}) {
    // 构建URL
    const url = new URL(window.location.href);

    // 如果是系统配置页面，只保留panel参数
    if (panel === 'system-config') {
        url.search = ''
        url.searchParams.set('panel', panel);
    } else {
        // 其他页面保留所有参数
        url.searchParams.set('panel', panel);

        // 添加其他参数
        for (const key in params) {
            if (params[key]) {
                url.searchParams.set(key, params[key]);
            } else {
                url.searchParams.delete(key);
            }
        }
    }

    // 更新历史记录并跳转
    window.history.pushState({}, '', url.toString());

    // 显示对应面板
    showPanel(panel);

    // 收起移动端菜单
    collapseNavbarMenu();
}

// 收起移动端导航菜单
function collapseNavbarMenu() {
    const navbarCollapse = document.getElementById('navbarNav');
    // 检查导航栏是否处于展开状态
    if (navbarCollapse && navbarCollapse.classList.contains('show')) {
        // Bootstrap 5方式收起导航栏
        const bsCollapse = new bootstrap.Collapse(navbarCollapse);
        bsCollapse.hide();
    }
}

// 显示面板
function showPanel(panel) {
    // 隐藏所有面板
    tasksPanel.style.display = 'none';
    recordsPanel.style.display = 'none';
    if (configsPanel) configsPanel.style.display = 'none';
    systemConfigPanel.style.display = 'none';

    // 移除所有导航激活状态
    navTasks.classList.remove('active');
    navRecords.classList.remove('active');
    navSystemConfig.classList.remove('active');

    // 从URL获取参数
    const url = new URL(window.location.href);
    const page = parseInt(url.searchParams.get('page')) || 1;
    const pageSize = parseInt(url.searchParams.get('pageSize')) || 10;
    const taskId = url.searchParams.get('taskId') || '';

    // 根据选择显示对应面板
    if (panel === 'tasks') {
        tasksPanel.style.display = 'block';
        navTasks.classList.add('active');
        loadTasks(true); // 显示加载动画
    } else if (panel === 'records') {
        recordsPanel.style.display = 'block';
        navRecords.classList.add('active');
        loadRecords(page, pageSize, taskId, true); // 显示加载动画
    } else if (panel === 'system-config') {
        systemConfigPanel.style.display = 'block';
        navSystemConfig.classList.add('active');
        loadSystemConfig();
    }
}

// 切换存储配置面板
function toggleStorageConfigPanels() {
    const type = storageType.value;

    if (type === 'local') {
        localStorageConfig.style.display = 'block';
        s3StorageConfig.style.display = 'none';
    } else if (type === 's3') {
        localStorageConfig.style.display = 'none';
        s3StorageConfig.style.display = 'block';
    }
}

// 加载系统配置
function loadSystemConfig() {
    // 显示加载动画
    showSectionLoading('system-config-panel');

    // 加载配置
    apiRequest('/api/configs')
        .then(result => {
            if (result.code === 200) {
                const configs = result.data;
                
                // 处理配置
                configs.forEach(config => {
                    try {
                        switch (config.configKey) {
                            case 'storage.type':
                                document.getElementById('storage-type').value = config.configValue;
                                toggleStorageConfigPanels();
                                break;
                            case 'storage.localPath':
                                document.getElementById('local-path').value = config.configValue;
                                break;
                            case 'storage.s3Endpoint':
                                document.getElementById('s3-endpoint').value = config.configValue;
                                break;
                            case 'storage.s3Region':
                                document.getElementById('s3-region').value = config.configValue;
                                break;
                            case 'storage.s3AccessKey':
                                document.getElementById('s3-access-key').value = config.configValue;
                                break;
                            case 'storage.s3SecretKey':
                                document.getElementById('s3-secret-key').value = config.configValue;
                                break;
                            case 'storage.s3Bucket':
                                document.getElementById('s3-bucket').value = config.configValue;
                                break;
                            case 'webhook.enabled':
                                document.getElementById('webhook-enabled').checked = config.configValue === 'true';
                                updateWebhookFormFields();
                                break;
                            case 'webhook.url':
                                document.getElementById('webhook-url').value = config.configValue;
                                break;
                            case 'webhook.headers':
                                document.getElementById('webhook-headers').value = config.configValue;
                                break;
                            case 'webhook.body':
                                document.getElementById('webhook-body').value = config.configValue;
                                break;
                            case 'system.autoCleanupDays':
                                document.getElementById('auto-cleanup-days').value = config.configValue;
                                break;
                            case 'system.siteName':
                                document.getElementById('site-name').value = config.configValue;
                                break;
                        }
                    } catch (error) {
                        console.error(`加载配置项${config.configKey}时出错:`, error);
                    }
                });
                
                // 获取系统版本并显示在只读框中
                fetch('/api/info')
                    .then(response => response.json())
                    .then(result => {
                        if (result.code === 200 && result.data.version) {
                            document.getElementById('site-version').value = result.data.version;
                        }
                    })
                    .catch(error => {
                        console.error('获取版本信息失败:', error);
                    });

                // 完成加载
                hideSectionLoading('system-config-panel');
            } else {
                showToast('加载配置失败: ' + result.msg, 'danger');
                hideSectionLoading('system-config-panel');
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showToast('加载配置失败: ' + error.message, 'danger');
            hideSectionLoading('system-config-panel');
        });
}

// 更新Webhook表单字段禁用状态
function updateWebhookFormFields() {
    const webhookEnabled = document.getElementById('webhook-enabled').checked;
    const webhookFields = [
        'webhook-url',
        'webhook-headers',
        'webhook-body'
    ];

    // 根据启用状态更新表单字段禁用状态
    webhookFields.forEach(fieldId => {
        const field = document.getElementById(fieldId);
        if (field) {
            field.disabled = !webhookEnabled;
        }
    });

    // 测试按钮
    const testButton = document.getElementById('btn-test-webhook');
    if (testButton) {
        testButton.disabled = !webhookEnabled;
    }
}

// 保存系统配置
function saveSystemConfig() {
    const storageType = document.getElementById('storage-type').value;
    const localPath = document.getElementById('local-path').value;
    const s3Endpoint = document.getElementById('s3-endpoint').value;
    const s3Region = document.getElementById('s3-region').value;
    const s3AccessKey = document.getElementById('s3-access-key').value;
    const s3SecretKey = document.getElementById('s3-secret-key').value;
    const s3Bucket = document.getElementById('s3-bucket').value;
    const webhookEnabled = document.getElementById('webhook-enabled').checked;
    const webhookUrl = document.getElementById('webhook-url').value;
    const webhookHeaders = document.getElementById('webhook-headers').value;
    const webhookBody = document.getElementById('webhook-body').value;
    const password = document.getElementById('system-password').value;
    const confirmPassword = document.getElementById('confirm-password').value;
    const autoCleanupDays = document.getElementById('auto-cleanup-days').value;
    const siteName = document.getElementById('site-name').value;

    // 检查密码是否匹配
    if (password !== confirmPassword) {
        showToast('两次输入的密码不一致', 'warning');
        return;
    }

    // siteName
    if (siteName && siteName.length > 8) {
        showToast('站点名称最多支持8个字符', 'warning');
        return;
    }

    // 构建配置
    const configs = [
        {
            configKey: 'storage.type',
            configValue: storageType,
            description: '存储类型'
        },
        {
            configKey: 'storage.localPath',
            configValue: localPath,
            description: '本地存储路径'
        },
        {
            configKey: 'storage.s3Endpoint',
            configValue: s3Endpoint,
            description: 'S3端点'
        },
        {
            configKey: 'storage.s3Region',
            configValue: s3Region,
            description: 'S3区域'
        },
        {
            configKey: 'storage.s3AccessKey',
            configValue: s3AccessKey,
            description: 'S3访问密钥'
        },
        {
            configKey: 'storage.s3SecretKey',
            configValue: s3SecretKey,
            description: 'S3私有密钥'
        },
        {
            configKey: 'storage.s3Bucket',
            configValue: s3Bucket,
            description: 'S3存储桶'
        },
        {
            configKey: 'webhook.enabled',
            configValue: webhookEnabled ? 'true' : 'false',
            description: '是否启用Webhook通知'
        },
        {
            configKey: 'webhook.url',
            configValue: webhookUrl,
            description: 'Webhook URL'
        },
        {
            configKey: 'webhook.headers',
            configValue: webhookHeaders,
            description: 'Webhook请求头'
        },
        {
            configKey: 'webhook.body',
            configValue: webhookBody,
            description: 'Webhook请求体模板'
        },
        {
            configKey: 'system.autoCleanupDays',
            configValue: autoCleanupDays,
            description: '自动清理时间（天）'
        },
        {
            configKey: 'system.siteName',
            configValue: siteName,
            description: '站点名称'
        }
    ];

    // 如果设置了新密码，则添加到配置中
    if (password) {
        configs.push({
            configKey: 'system.password',
            configValue: password,
            description: '系统登录密码'
        });
    }

    // 显示加载动画
    showLoading('正在保存配置...');

    // 调用API保存配置
    saveConfigBatch(configs)
        .then(result => {
            hideLoading();

            if (result === true) {
                showToast('配置保存成功', 'success');
                
                // 刷新站点信息，立即显示更新后的站点名称
                getSiteInfo();

                // 如果设置了密码，提示用户
                if (password && !localStorage.getItem('backupSystemAuth')) {
                    Swal.fire({
                        title: '已设置密码',
                        text: '系统将在下次访问时要求输入密码进行登录',
                        icon: 'info',
                        confirmButtonText: '确定'
                    });
                }
            } else {
                showToast('保存失败：保存过程出现错误', 'danger');
            }
        })
        .catch(error => {
            hideLoading();
            console.error('Error:', error);
            showToast(`保存失败: ${error.message}`, 'danger');
        });
}

// 批量保存配置
async function saveConfigBatch(configs) {
    try {

        // 先获取所有现有配置
        const result = await apiRequest('/api/configs');
        if (!result) {
            throw new Error('获取现有配置失败');
        }

        const existingConfigs = result.code === 200 ? (result.data || []) : [];

        // 针对每个配置项进行保存
        for (const config of configs) {

            // 检查是否已存在
            const existingConfig = existingConfigs.find(c => c.configKey === config.configKey);

            if (existingConfig) {
                // 更新已有配置
                const updateResult = await apiRequest(`/api/configs/update?id=${existingConfig.id}`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        id: existingConfig.id,
                        configKey: config.configKey,
                        configValue: config.configValue,
                        description: config.description
                    })
                }, false, '');

                if (!updateResult || updateResult.code !== 200) {
                    console.error('更新配置失败:', updateResult);
                    throw new Error(`更新配置 ${config.configKey} 失败: ${updateResult ? updateResult.msg : '未知错误'}`);
                }

            } else {
                // 创建新配置
                const createResult = await apiRequest('/api/configs', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(config)
                });

                if (!createResult || createResult.code !== 200) {
                    console.error('创建配置失败:', createResult);
                    throw new Error(`创建配置 ${config.configKey} 失败: ${createResult ? createResult.msg : '未知错误'}`);
                }

            }
        }

        return true;
    } catch (error) {
        console.error('保存配置失败:', error);
        throw error;
    }
}

// 加载任务列表
function loadTasks(showLoadingIndicator = true) {
    // 使用apiRequest函数传递加载文本和是否显示加载动画的参数
    apiRequest('/api/tasks', {}, showLoadingIndicator, '正在加载...')
        .then(result => {
            if (!result) return;

            if (result.code === 200) {
                // 处理不同的返回格式
                let tasks = [];
                if (Array.isArray(result.data)) {
                    tasks = result.data;
                } else if (result.data && result.data.list) {
                    tasks = result.data.list;
                } else if (result.data && result.data.tasks) {
                    tasks = result.data.tasks;
                } else if (result.data) {
                    tasks = result.data;
                }

                tasksList = tasks || [];
                renderTaskList(tasksList);

                // 更新任务筛选器
                updateTaskFilter();

                // 显示成功消息
            } else {
                showToast(`加载任务列表失败：${result.msg}`, 'danger');
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showToast(`加载任务列表失败：${error.message}`, 'danger');
        });
}

// 渲染任务列表
function renderTaskList(tasks) {
    tasksTableBody.innerHTML = '';

    // 确保tasks是数组
    if (!Array.isArray(tasks)) {
        tasksTableBody.innerHTML = '<tr><td colspan="7" class="text-center text-danger">任务数据格式错误</td></tr>';
        showToast('任务数据格式错误', 'danger');
        return;
    }

    if (tasks.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = '<td colspan="7" class="text-center">暂无任务</td>';
        tasksTableBody.appendChild(tr);
        return;
    }

    tasks.forEach(task => {
        try {
            const tr = document.createElement('tr');

            // 获取下一次执行时间，从extraData中读取
            let nextExecutionTime = '-';
            if (task.enabled) {
                if (task.extraData && task.extraData.nextExecutionTime) {
                    nextExecutionTime = task.extraData.nextExecutionTime;
                } else {
                    nextExecutionTime = '<small><i>计算中...</i></small>';
                }
            } else {
                nextExecutionTime = '任务已禁用';
            }

            // 安全处理任务名称等属性
            const taskName = escapeHtml(task.name || '未命名任务');
            const taskType = getBackupTypeName(task.type || 'unknown');
            const taskSchedule = escapeHtml(task.schedule || '-');

            tr.innerHTML = `
            <td>${task.id}</td>
                <td>
                    <div class="d-flex align-items-center">
                        <span>${taskName}</span>
                    </div>
                </td>
                <td>${taskType}</td>
                <td>${taskSchedule}</td>
                <td id="next-time-${task.id}">${nextExecutionTime}</td>
            <td>
                <div class="form-check form-switch ${task.enabled ? 'task-enabled' : 'task-disabled'}">
                    <input class="form-check-input task-enabled-switch" type="checkbox" role="switch" 
                        id="enabledSwitch-${task.id}" ${task.enabled ? 'checked' : ''} data-id="${task.id}" data-enabled="${task.enabled}">
                </div>
            </td>
            <td>
                <div class="task-buttons-container">
                <button class="btn btn-sm btn-primary btn-icon btn-edit" data-id="${task.id}">编辑</button>
                    <button class="btn btn-sm btn-success btn-icon btn-execute" data-id="${task.id}" ${!task.enabled ? 'disabled' : ''}>执行</button>
                <button class="btn btn-sm btn-info btn-icon btn-records" data-id="${task.id}">记录</button>
                <button class="btn btn-sm btn-danger btn-icon btn-delete" data-id="${task.id}">删除</button>
                </div>
            </td>
        `;

            tasksTableBody.appendChild(tr);
        } catch (error) {
            console.error('渲染任务时出错:', error, task);
        }
    });

    // 绑定按钮事件
    document.querySelectorAll('.btn-edit').forEach(btn => {
        btn.addEventListener('click', function () {
            const id = parseInt(this.dataset.id);
            editTask(id);
        });
    });

    document.querySelectorAll('.btn-execute').forEach(btn => {
        btn.addEventListener('click', function () {
            const id = parseInt(this.dataset.id);
            executeTask(id);
        });
    });

    document.querySelectorAll('.btn-records').forEach(btn => {
        btn.addEventListener('click', function () {
            const id = parseInt(this.dataset.id);
            showTaskRecords(id);
        });
    });

    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', function () {
            const id = parseInt(this.dataset.id);
            deleteTask(id);
        });
    });

    // 绑定启用/禁用开关事件
    document.querySelectorAll('.task-enabled-switch').forEach(switchEl => {
        // 添加事件监听
        switchEl.addEventListener('change', function () {
            const id = parseInt(this.dataset.id);
            const enabled = this.checked;
            updateTaskEnabled(id, enabled);
        });
    });
}

// 更新任务启用状态
function updateTaskEnabled(id, enabled) {
    // 获取当前任务
    const task = tasksList.find(t => t.id === id);
    if (!task) {
        showToast('任务不存在', 'danger');
        return;
    }

    // 如果状态没有变化，不需要更新
    if (task.enabled === enabled) {
        return;
    }

    // 禁用所有开关，防止重复操作
    const allSwitches = document.querySelectorAll('.task-enabled-switch');
    allSwitches.forEach(sw => sw.disabled = true);

    // 显示区域加载状态，而不是全局加载动画
    showSectionLoading('tasks-panel');

    // 调用API更新任务状态，使用false参数避免显示全局加载动画
    apiRequest(`/api/tasks/updateEnabled?id=${id}&enabled=${enabled}`, {
        method: 'POST'
    }, false)  // 不显示全局加载动画，使用区域加载代替
        .then(result => {
            if (!result) return;

            if (result.code === 200) {
                // 更新本地状态
                task.enabled = enabled;

                // 更新开关容器的样式类
                const switchContainer = document.querySelector(`#enabledSwitch-${id}`).closest('.form-check');
                if (switchContainer) {
                    if (enabled) {
                        switchContainer.classList.remove('task-disabled');
                        switchContainer.classList.add('task-enabled');
                    } else {
                        switchContainer.classList.remove('task-enabled');
                        switchContainer.classList.add('task-disabled');
                    }
                }

                // 显示成功消息
                showToast(`任务已${enabled ? '启用' : '禁用'}成功！`, 'success');

                // 成功后直接加载任务列表，不需要延时
                loadTasks(false); // 重新加载任务列表，不显示全局加载动画
            } else {
                // 恢复开关状态
                const switchEl = document.getElementById(`enabledSwitch-${id}`);
                if (switchEl) {
                    switchEl.checked = !enabled;
                }

                // 显示错误消息
                showToast(`修改状态失败: ${result.msg}`, 'danger');
            }
        })
        .catch(error => {
            console.error('Error:', error);

            // 恢复开关状态
            const switchEl = document.getElementById(`enabledSwitch-${id}`);
            if (switchEl) {
                switchEl.checked = !enabled;
            }

            // 显示错误消息
            showToast(`修改状态失败: ${error.message}`, 'danger');
        })
        .finally(() => {
            // 隐藏加载状态
            hideSectionLoading('tasks-panel');

            // 恢复所有开关
            allSwitches.forEach(sw => sw.disabled = false);
        });
}

// 加载备份记录列表
function loadRecords(page = 1, pageSize = 10, taskId = null, showLoadingIndicator = true) {
    // 记录当前页码和任务ID，方便删除后刷新
    currentPage = page;
    currentTaskId = taskId;

    showSectionLoading('records-panel');

    // 每次加载备份记录时，重新加载任务列表
    apiRequest('/api/tasks', {}, false, '正在加载...')
        .then(result => {
            if (result && result.code === 200) {
                // 处理不同的返回格式
                let tasks = [];
                if (Array.isArray(result.data)) {
                    tasks = result.data;
                } else if (result.data && result.data.list) {
                    tasks = result.data.list;
                } else if (result.data && result.data.tasks) {
                    tasks = result.data.tasks;
                } else if (result.data) {
                    tasks = result.data;
                }

                tasksList = tasks || [];

                // 更新任务筛选器并设置当前选中的任务
                updateTaskFilter(taskId);

                // 继续加载记录
                loadRecordsData(page, pageSize, taskId, showLoadingIndicator);
            } else {
                showToast('无法加载任务列表，筛选功能可能不可用', 'warning');
                loadRecordsData(page, pageSize, taskId, showLoadingIndicator);
            }
        })
        .catch(error => {
            console.error('加载任务列表失败:', error);
            showToast('无法加载任务列表，筛选功能可能不可用', 'warning');
            loadRecordsData(page, pageSize, taskId, showLoadingIndicator);
        });
}

// 实际加载备份记录数据的函数
function loadRecordsData(page = 1, pageSize = 10, taskId = null, showLoadingIndicator = true) {
    // 构建API请求URL
    let url = `/api/records?page=${page}&pageSize=${pageSize}`;
    if (taskId) {
        url = `/api/records/task?taskId=${taskId}&page=${page}&pageSize=${pageSize}`;
    }

    // 使用apiRequest函数传递加载文本
    apiRequest(url, {}, showLoadingIndicator, '正在加载...')
        .then(result => {
            if (result.code === 200) {
                // 处理不同的返回格式
                let records = [];
                let total = 0;

                if (Array.isArray(result.data)) {
                    records = result.data;
                    total = result.total || records.length;
                } else if (result.data && result.data.records) {
                    records = result.data.records;
                    total = result.data.total || records.length;
                } else if (result.data && result.data.list) {
                    records = result.data.list;
                    total = result.data.total || records.length;
                } else if (result.data) {
                    records = result.data;
                    total = result.total || records.length;
                }

                // 渲染记录列表
                renderRecords(records);

                // 生成分页
                renderPagination(total, page, pageSize, taskId);

                // 显示成功消息
            } else {
                showToast(`加载备份记录失败：${result.msg}`, 'danger');
                recordsTableBody.innerHTML = `<tr><td colspan="8" class="text-center text-danger">加载失败: ${result.msg}</td></tr>`;
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showToast(`加载备份记录失败：${error.message}`, 'danger');
            recordsTableBody.innerHTML = `<tr><td colspan="8" class="text-center text-danger">加载失败: ${error.message}</td></tr>`;
        })
        .finally(() => {
            hideSectionLoading('records-panel');
        });
}

// 渲染备份记录
function renderRecords(records) {
    recordsTableBody.innerHTML = '';

    // 确保records是数组
    if (!Array.isArray(records)) {
        console.error('记录列表不是数组:', records);
        recordsTableBody.innerHTML = '<tr><td colspan="8" class="text-center text-danger">记录数据格式错误</td></tr>';
        showToast('记录数据格式错误', 'danger');
        return;
    }

    if (records.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = '<td colspan="8" class="text-center">暂无记录</td>';
        recordsTableBody.appendChild(tr);
        return;
    }

    records.forEach(record => {
        try {
            // 处理状态
            const status = getStatusName(record.status || 'unknown');
            const statusClass = `status-${record.status || 'unknown'}`;
            const taskName = escapeHtml(record.taskName || '未知任务');

            // 计算执行时间
            let executionTime = '-';
            if (record.startTime) {
                try {
                    const start = new Date(record.startTime);

                    // 检查任务是否正在执行中
                    if (!record.endTime || new Date(record.endTime).getFullYear() <= 1970 || record.status === 'running') {
                        // 任务正在执行中，使用当前时间计算
                        const now = new Date();
                        const diff = now - start;

                        if (!isNaN(diff)) {
                            // 计算时间差
                            const seconds = Math.floor(diff / 1000);
                            const minutes = Math.floor(seconds / 60);
                            const hours = Math.floor(minutes / 60);

                            // 显示"执行中"标识
                            if (hours > 0) {
                                executionTime = `${hours}小时${minutes % 60}分钟 (执行中)`;
                            } else if (minutes > 0) {
                                executionTime = `${minutes}分钟${seconds % 60}秒 (执行中)`;
                            } else {
                                executionTime = `${seconds}秒 (执行中)`;
                            }
                        } else {
                            executionTime = '执行中';
                        }
                    } else {
                        // 任务已完成，使用结束时间计算
                        const end = new Date(record.endTime);
                        const diff = end - start;

                        if (isNaN(diff)) {
                            executionTime = '-';
                        } else {
                            // 计算时间差
                            const seconds = Math.floor(diff / 1000);
                            const minutes = Math.floor(seconds / 60);
                            const hours = Math.floor(minutes / 60);

                            // 根据时间长度选择合适的显示单位
                            if (hours > 0) {
                                executionTime = `${hours}小时${minutes % 60}分钟`;
                            } else if (minutes > 0) {
                                executionTime = `${minutes}分钟${seconds % 60}秒`;
                            } else {
                                executionTime = `${seconds}秒`;
                            }
                        }
                    }
                } catch (timeError) {
                    console.error('计算执行时间出错:', timeError);
                    executionTime = '-';
                }
            }

            const tr = document.createElement('tr');

            tr.innerHTML = `
            <td>${record.id}</td>
                <td>
                    <div class="d-flex align-items-center">
                        <span>${taskName}</span>
                        </div>
                    </td>
                <td>${formatDateTime(record.startTime)}</td>
                    <td>${(!record.endTime || new Date(record.endTime).getFullYear() <= 1970 || record.status === 'running') ? '<span class="text-muted">执行中</span>' : formatDateTime(record.endTime)}</td>
                <td>${executionTime}</td>
                <td><span class="badge ${statusClass}">${status}</span></td>
                <td>${record.fileSize ? formatFileSize(record.fileSize) : '-'}</td>
                <td>
                    <div class="record-buttons-container">
                        <button class="btn btn-sm btn-primary btn-icon btn-view-record" data-id="${record.id}">查看</button>
                        ${record.filePath && record.status !== 'cleaned' ? `<button class="btn btn-sm btn-success btn-icon btn-download" data-id="${record.id}">下载</button>` : ''}
                        <button class="btn btn-sm btn-danger btn-icon btn-delete-record" data-id="${record.id}">删除</button>
                    </div>
                </td>
            `;

            recordsTableBody.appendChild(tr);
        } catch (error) {
            console.error('渲染记录时出错:', error, record);
        }
    });

    // 绑定按钮事件
    document.querySelectorAll('.btn-view-record').forEach(btn => {
        btn.addEventListener('click', function () {
            const id = parseInt(this.dataset.id);
            showRecordDetails(id);
        });
    });

    document.querySelectorAll('.btn-download').forEach(btn => {
        btn.addEventListener('click', function () {
            const id = parseInt(this.dataset.id);
            downloadRecord(id);
        });
    });

    document.querySelectorAll('.btn-delete-record').forEach(btn => {
        btn.addEventListener('click', function () {
            const id = parseInt(this.dataset.id);
            deleteRecord(id);
        });
    });
}

// 渲染分页
function renderPagination(total, currentPage, pageSize, taskId = '') {
    const totalPages = Math.ceil(total / pageSize);
    pagination.innerHTML = '';

    if (totalPages <= 1) {
        return;
    }

    // 上一页按钮
    const prevLi = document.createElement('li');
    prevLi.className = `page-item ${currentPage === 1 ? 'disabled' : ''}`;
    const prevA = document.createElement('a');
    prevA.className = 'page-link';
    prevA.href = '#';
    prevA.textContent = '上一页';
    prevA.addEventListener('click', function (e) {
        e.preventDefault();
        navigateTo('records', {page: currentPage - 1, pageSize, taskId});
    });
    prevLi.appendChild(prevA);
    pagination.appendChild(prevLi);

    // 页码按钮
    let startPage = Math.max(1, currentPage - 2);
    let endPage = Math.min(totalPages, startPage + 4);

    if (endPage - startPage < 4) {
        startPage = Math.max(1, endPage - 4);
    }

    for (let i = startPage; i <= endPage; i++) {
        const pageLi = document.createElement('li');
        pageLi.className = `page-item ${i === currentPage ? 'active' : ''}`;
        const pageA = document.createElement('a');
        pageA.className = 'page-link';
        pageA.href = '#';
        pageA.textContent = i;
        pageA.addEventListener('click', function (e) {
            e.preventDefault();
            navigateTo('records', {page: i, pageSize, taskId});
        });
        pageLi.appendChild(pageA);
        pagination.appendChild(pageLi);
    }

    // 下一页按钮
    const nextLi = document.createElement('li');
    nextLi.className = `page-item ${currentPage === totalPages ? 'disabled' : ''}`;
    const nextA = document.createElement('a');
    nextA.className = 'page-link';
    nextA.href = '#';
    nextA.textContent = '下一页';
    nextA.addEventListener('click', function (e) {
        e.preventDefault();
        if (currentPage < totalPages) {
            navigateTo('records', {page: currentPage + 1, pageSize, taskId});
        }
    });
    nextLi.appendChild(nextA);
    pagination.appendChild(nextLi);
}

// 更新任务筛选器
function updateTaskFilter(currentTaskId = null) {
    // 清空选项
    filterTask.innerHTML = '<option value="">所有任务</option>';

    // 添加任务选项
    tasksList.forEach(task => {
        const option = document.createElement('option');
        option.value = task.id;
        option.textContent = task.name;
        filterTask.appendChild(option);
    });

    // 设置当前选中的任务ID
    if (currentTaskId) {
        // 检查选中的值是否在任务列表中
        const taskExists = tasksList.some(task => task.id.toString() === currentTaskId.toString());
        if (taskExists) {
            filterTask.value = currentTaskId;
        } else {
            filterTask.value = '';
        }
    } else {
        // 从URL中获取taskId参数
        const url = new URL(window.location.href);
        const urlTaskId = url.searchParams.get('taskId');
        if (urlTaskId) {
            const taskExists = tasksList.some(task => task.id.toString() === urlTaskId);
            if (taskExists) {
                filterTask.value = urlTaskId;
            } else {
                filterTask.value = '';
            }
        } else {
            filterTask.value = '';
        }
    }
}

// 编辑任务
function editTask(id) {
    // 找到任务对象
    const task = tasksList.find(task => task.id === id);
    if (!task) {
        showToast('任务不存在', 'danger');
        return;
    }

    // 设置当前任务ID，标记为编辑模式
    currentTaskId = id;

    // 设置表单值
    document.getElementById('task-id').value = task.id;
    document.getElementById('task-name').value = task.name;
    document.getElementById('task-type').value = task.type;
    document.getElementById('task-schedule').value = task.schedule;
    document.getElementById('task-enabled').checked = task.enabled;

    // 解析源信息
    let sourceInfo = {};
    try {
        sourceInfo = JSON.parse(task.sourceInfo);
    } catch (error) {
        console.error('解析源信息失败:', error);
    }

    // 设置特定类型的表单字段
    if (task.type === 'database') {
        document.getElementById('db-type').value = sourceInfo.type || 'mysql';
        document.getElementById('db-host').value = sourceInfo.host || 'localhost';
        document.getElementById('db-port').value = sourceInfo.port || 3306;
        document.getElementById('db-user').value = sourceInfo.user || 'root';
        document.getElementById('db-password').value = sourceInfo.password || '';
        document.getElementById('db-name').value = sourceInfo.database || '';
    } else if (task.type === 'file') {
        document.getElementById('file-paths').value = sourceInfo.paths ? sourceInfo.paths.join('\n') : '';
    }

    // 切换配置面板
    toggleConfigPanels();

    // 设置模态框标题
    document.getElementById('task-modal-title').textContent = '编辑任务';

    // 显示模态框
    showModal('task-modal');
}

// 执行任务
function executeTask(id) {
    // 获取当前任务
    const task = tasksList.find(t => t.id === id);
    if (!task) {
        showToast('任务不存在', 'danger');
        return;
    }

    // 显示消息提示用户正在执行
    showToast('准备执行任务，请稍候...', 'info');

    // 禁用执行按钮
    const executeButton = document.querySelector(`.btn-execute[data-id="${id}"]`);
    if (executeButton) {
        executeButton.disabled = true;
        executeButton.textContent = '执行中...';
    }

    // 显示区域加载状态，而不是全局加载动画
    showSectionLoading('tasks-panel');

    // 调用API执行任务，不使用全局加载动画
    apiRequest(`/api/tasks/execute?id=${id}`, {
        method: 'POST'
    }, false)
        .then(result => {
            if (result.code === 200) {
                showToast('任务已提交执行，请稍后查看执行结果', 'success');

                // 3秒后重新加载任务列表，显示最新状态
                setTimeout(() => {
                    loadTasks(false); // 不显示全局加载动画
                }, 1000);
            } else {
                showToast(`执行失败: ${result.msg}`, 'danger');

                // 恢复按钮状态
                if (executeButton) {
                    executeButton.disabled = false;
                    executeButton.textContent = '执行';
                }
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showToast(`执行失败: ${error.message}`, 'danger');

            // 恢复按钮状态
            if (executeButton) {
                executeButton.disabled = false;
                executeButton.textContent = '执行';
            }
        })
        .finally(() => {
            // 隐藏区域加载状态
            hideSectionLoading('tasks-panel');
        });
}

// 删除任务
function deleteTask(id) {
    Swal.fire({
        title: '确认删除',
        text: '确定要删除此任务吗？此操作无法撤销！',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        confirmButtonColor: '#d33',
        cancelButtonColor: '#3085d6'
    }).then((result) => {
        if (result.isConfirmed) {
            // 显示操作提示
            // showToast('正在删除任务...', 'info');

            // 显示区域加载
            showSectionLoading('tasks-panel');

            // 找到当前行并添加淡出效果
            const taskRow = document.querySelector(`tr:has(button.btn-delete[data-id="${id}"])`);
            if (taskRow) {
                taskRow.style.transition = 'opacity 0.5s ease';
                taskRow.style.opacity = '0.5';
            }

            apiRequest(`/api/tasks/delete?id=${id}`, {
                method: 'POST'
            }, false) // 不使用全局加载动画
                .then(result => {
                    if (result.code === 200) {
                        // 任务被成功删除，可以在UI中立即反映
                        if (taskRow) {
                            taskRow.style.opacity = '0';
                            setTimeout(() => {
                                if (taskRow.parentNode) {
                                    taskRow.parentNode.removeChild(taskRow);
                                }
                            }, 500);
                        }

                        showToast('任务已成功删除', 'success');

                        // 稍微延迟后重新加载任务列表，确保用户可以看到成功消息
                        setTimeout(() => {
                            loadTasks(false); // 重新加载任务列表，不显示全局加载动画
                        }, 800);
                    } else {
                        // 恢复行的显示
                        if (taskRow) {
                            taskRow.style.opacity = '1';
                        }

                        showToast(`删除失败: ${result.msg}`, 'danger');
                    }
                })
                .catch(error => {
                    // 恢复行的显示
                    if (taskRow) {
                        taskRow.style.opacity = '1';
                    }

                    console.error('Error:', error);
                    showToast(`删除失败: ${error.message}`, 'danger');
                })
                .finally(() => {
                    // 隐藏区域加载
                    hideSectionLoading('tasks-panel');
                });
        }
    });
}

// 显示任务记录
function showTaskRecords(id) {
    // 使用URL参数导航而不使用localStorage
    navigateTo('records', {taskId: id, page: 1});
}

// 显示记录详情
function showRecordDetails(id) {
    // 获取记录详情并显示
    apiRequest(`/api/records/get?id=${id}`, {}, false, '')
        .then(result => {
            if (!result) return;

            if (result.code === 200) {
                const record = result.data;
                Swal.fire({
                    title: '备份记录详情',
                    html: `
                        <div style="text-align: left">
                            <p><strong>状态:</strong> ${getStatusName(record.status)}</p>
                            <p><strong>开始时间:</strong> ${formatDateTime(record.startTime)}</p>
                            <p><strong>结束时间:</strong> ${(!record.endTime || new Date(record.endTime).getFullYear() <= 1970 || record.status === 'running') ? '执行中' : formatDateTime(record.endTime)}</p>
                            <p><strong>文件大小:</strong> ${record.fileSize ? formatFileSize(record.fileSize) : '无文件'}</p>
                            <p><strong>文件路径:</strong> ${record.filePath || '无文件'}</p>
                            <p><strong>错误信息:</strong> ${record.errorMessage || '无错误'}</p>
                        </div>
                    `,
                    icon: 'info',
                    confirmButtonText: '关闭'
                });
            } else {
                showToast('加载记录详情失败: ' + result.msg, 'danger');
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showToast('加载记录详情失败: ' + error.message, 'danger');
        });
}

// 保存任务
function saveTask() {
    try {

        // 获取表单数据
        const id = document.getElementById('task-id').value;
        const name = document.getElementById('task-name').value;
        const type = document.getElementById('task-type').value;
        const schedule = document.getElementById('task-schedule').value;
        const enabled = document.getElementById('task-enabled').checked;

        // 根据类型获取源信息
        let sourceInfo = {};
        if (type === 'database') {
            sourceInfo = {
                type: document.getElementById('db-type').value,
                host: document.getElementById('db-host').value,
                port: parseInt(document.getElementById('db-port').value) || 3306,
                user: document.getElementById('db-user').value,
                password: document.getElementById('db-password').value,
                database: document.getElementById('db-name').value
            };
        } else if (type === 'file') {
            const paths = document.getElementById('file-paths').value
                .split('\n')
                .map(p => p.trim())
                .filter(p => p);

            sourceInfo = {
                paths: paths
            };
        }

        // 表单验证
        if (!name) {
            showToast('请输入任务名称', 'warning');
            return;
        }

        if (!schedule) {
            showToast('请输入Cron表达式', 'warning');
            return;
        }

        // 验证Cron表达式
        const cronValidation = validateCronExpression(schedule);
        if (!cronValidation.valid) {
            showToast(cronValidation.message, 'warning');
            return;
        }

        if (type === 'database') {
            if (!sourceInfo.host || !sourceInfo.user) {
                showToast('请完整填写数据库配置', 'warning');
                return;
            }
        } else if (type === 'file') {
            if (!sourceInfo.paths || sourceInfo.paths.length === 0) {
                showToast('请输入至少一个文件路径', 'warning');
                return;
            }
        }

        // 构建任务对象
        const task = {
            name,
            type,
            schedule,
            enabled,
            sourceInfo: JSON.stringify(sourceInfo)
        };

        // 如果有ID，表示是更新
        if (id) {
            task.id = parseInt(id);
        }


        // 禁用保存按钮，防止重复提交
        const saveButton = document.getElementById('btn-save-task');
        if (saveButton) {
            saveButton.disabled = true;
            saveButton.textContent = '保存中...';
        }

        // 显示加载动画
        showLoading('正在保存任务...');

        // 确定API地址
        const apiUrl = id ? `/api/tasks/update?id=${id}` : '/api/tasks';

        // 调用API保存任务
        apiRequest(apiUrl, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(task)
        })
            .then(result => {
                hideLoading();

                // 恢复保存按钮状态
                if (saveButton) {
                    saveButton.disabled = false;
                    saveButton.textContent = '保存';
                }

                if (result && result.code === 200) {
                    // 关闭任务编辑模态框
                    closeModal('task-modal');

                    // 显示成功消息
                    showToast('任务保存成功', 'success');

                    // 重新加载任务列表
                    loadTasks(false);
                } else {
                    showToast(`保存失败: ${result ? result.msg : '未知错误'}`, 'danger');
                }
            })
            .catch(error => {
                hideLoading();
                console.error('保存任务时发生错误:', error);

                // 恢复保存按钮状态
                if (saveButton) {
                    saveButton.disabled = false;
                    saveButton.textContent = '保存';
                }

                showToast(`保存失败: ${error.message}`, 'danger');
            });
    } catch (error) {
        console.error('保存任务函数运行时错误:', error);

        // 恢复保存按钮状态
        const saveButton = document.getElementById('btn-save-task');
        if (saveButton) {
            saveButton.disabled = false;
            saveButton.textContent = '保存';
        }

        showToast(`操作失败: ${error.message}`, 'danger');
    }
}

// 重置任务表单
function resetTaskForm() {
    taskForm.reset();
    document.getElementById('task-id').value = '';
    document.getElementById('task-type').value = 'database';
    document.getElementById('task-enabled').checked = true;

    toggleConfigPanels();
}

// 切换配置面板
function toggleConfigPanels() {
    const type = document.getElementById('task-type').value;

    if (type === 'database') {
        document.getElementById('database-config').style.display = 'block';
        document.getElementById('file-config').style.display = 'none';
    } else if (type === 'file') {
        document.getElementById('database-config').style.display = 'none';
        document.getElementById('file-config').style.display = 'block';
    }
}

// 辅助函数
function getBackupTypeName(type) {
    const types = {
        'database': '数据库备份',
        'file': '文件备份'
    };
    return types[type] || type;
}

function getStatusName(status) {
    const statuses = {
        'pending': '等待中',
        'running': '执行中',
        'success': '成功',
        'failed': '失败',
        'cancelled': '已取消',
        'cleaned': '已清理'
    };
    return statuses[status] || status;
}

function formatDateTime(dateTimeStr) {
    if (!dateTimeStr) return '';
    const date = new Date(dateTimeStr);
    return date.toLocaleString('zh-CN');
}

function formatFileSize(size) {
    if (size < 1024) {
        return size + ' B';
    } else if (size < 1024 * 1024) {
        return (size / 1024).toFixed(2) + ' KB';
    } else if (size < 1024 * 1024 * 1024) {
        return (size / (1024 * 1024)).toFixed(2) + ' MB';
    } else {
        return (size / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
    }
}

function escapeHtml(html) {
    const div = document.createElement('div');
    div.textContent = html;
    return div.innerHTML;
}

// 显示提示消息
function showToast(message, type = 'info', duration = 1000) {
    if (!message) {
        console.warn('显示空消息');
        return;
    }

    // 处理错误对象
    if (message instanceof Error) {
        message = message.message || '发生错误';
        type = 'danger';
    }

    // 处理过长的消息
    if (typeof message === 'string' && message.length > 200) {
        console.log('完整消息:', message);
        message = message.substring(0, 200) + '...';
    }

    // 使用SweetAlert2显示消息提示（异步模式）
    const iconMap = {
        'success': 'success',
        'danger': 'error',
        'warning': 'warning',
        'info': 'info'
    };

    // 创建随机ID防止多个Toast重叠
    const toastId = `toast-${Date.now()}-${Math.random().toString(36).substring(2, 8)}`;

    // 使用mixin创建Toast，使其可以与其他Alert共存
    const Toast = Swal.mixin({
        toast: true,
        position: 'top-end',
        showConfirmButton: false,
        timer: duration,
        timerProgressBar: true,
        didOpen: (toast) => {
            toast.addEventListener('mouseenter', Swal.stopTimer);
            toast.addEventListener('mouseleave', Swal.resumeTimer);
        },
        customClass: {
            container: `toast-container-${toastId}`
        }
    });

    // 异步显示Toast，不阻止其他操作
    Toast.fire({
        icon: iconMap[type] || 'info',
        title: message
    });

    // 对于错误类型，同时在控制台输出
    if (type === 'danger' || type === 'error') {
        console.error('错误提示:', message);
    }
}

// 加载配置列表
function loadConfigs() {
    apiRequest('/api/configs')
        .then(result => {
            if (!result) return;

            if (result.code === 200) {
                const configs = result.data || [];
                renderConfigs(configs);
            } else {
                showToast('加载配置失败: ' + result.msg, 'danger');
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showToast('加载配置失败: ' + error.message, 'danger');
        });
}

// 渲染配置列表
function renderConfigs(configs) {
    configsTableBody.innerHTML = '';

    if (configs.length === 0) {
        const tr = document.createElement('tr');
        tr.innerHTML = '<td colspan="6" class="text-center">暂无配置</td>';
        configsTableBody.appendChild(tr);
        return;
    }

    configs.forEach(config => {
        const tr = document.createElement('tr');

        // 截断过长的配置值，仅在UI中显示前50个字符
        let displayValue = config.configValue;
        if (displayValue.length > 50) {
            displayValue = displayValue.substring(0, 50) + '...';
        }

        tr.innerHTML = `
            <td>${config.id}</td>
            <td>${escapeHtml(config.configKey)}</td>
            <td>${escapeHtml(displayValue)}</td>
            <td>${escapeHtml(config.description || '')}</td>
            <td>${formatDateTime(config.updatedAt)}</td>
            <td>
                <button class="btn btn-sm btn-primary btn-icon btn-edit-config" data-id="${config.id}">编辑</button>
                <button class="btn btn-sm btn-danger btn-icon btn-delete-config" data-id="${config.id}">删除</button>
            </td>
        `;

        configsTableBody.appendChild(tr);
    });

    // 绑定按钮事件
    document.querySelectorAll('.btn-edit-config').forEach(btn => {
        btn.addEventListener('click', function () {
            const id = parseInt(this.dataset.id);
            editConfig(id);
        });
    });

    document.querySelectorAll('.btn-delete-config').forEach(btn => {
        btn.addEventListener('click', function () {
            const id = parseInt(this.dataset.id);
            deleteConfig(id);
        });
    });
}

// 编辑配置
function editConfig(id) {

    // 获取配置详情
    fetch(`/api/configs/get?id=${id}`)
        .then(response => response.json())
        .then(result => {
            if (result.code === 200) {
                const config = result.data;

                // 填充表单
                document.getElementById('config-id').value = config.id;
                document.getElementById('config-key').value = config.configKey;
                document.getElementById('config-value').value = config.configValue;
                document.getElementById('config-description').value = config.description || '';

                // 显示模态框
                document.getElementById('config-modal-title').textContent = '编辑配置';
                showModal('config-modal');
            } else {
                showToast('加载配置详情失败: ' + result.msg, 'danger');
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showToast('加载配置详情失败: ' + error.message, 'danger');
        });
}

// 删除配置
function deleteConfig(id) {
    Swal.fire({
        title: '确认删除',
        text: '确定要删除此配置吗？',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        confirmButtonColor: '#d33',
        cancelButtonColor: '#3085d6'
    }).then((result) => {
        if (result.isConfirmed) {
            fetch(`/api/configs/delete?id=${id}`, {
                method: 'POST'
            })
                .then(response => response.json())
                .then(result => {
                    if (result.code === 200) {
                        showToast('配置删除成功', 'success');
                        loadConfigs();
                    } else {
                        showToast('删除配置失败: ' + result.msg, 'danger');
                    }
                })
                .catch(error => {
                    console.error('Error:', error);
                    showToast('删除配置失败: ' + error.message, 'danger');
                });
        }
    });
}

// 保存配置
function saveConfig() {
    // 显示加载动画
    showLoading('正在保存配置...');

    // 调用API保存配置
    apiRequest('/api/config/save', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(config)
    })
        .then(result => {
            hideLoading();

            if (result.code === 200) {
                showToast('配置保存成功', 'success');

                // 关闭配置模态框
                closeModal('configModal');

            } else {
                showToast(`保存失败: ${result.msg}`, 'danger');
            }
        })
        .catch(error => {
            hideLoading();
            console.error('Error:', error);
            showToast(`保存失败: ${error.message}`, 'danger');
        });
}

// 重置配置表单
function resetConfigForm() {
    configForm.reset();
    document.getElementById('config-id').value = '';
}

// API请求函数，自动添加授权头部并处理401错误
async function apiRequest(url, options = {}, showLoadingIndicator = true, loadingText = '处理中...') {
    try {
        // 显示加载动画（如果需要）
        if (showLoadingIndicator) {
            showLoading(loadingText);
        }

        // 确保options是一个对象
        if (typeof options === 'string') {
            // 如果options是字符串（如'POST'），将其转换为对象
            options = {
                method: options
            };
        }

        // 确保headers存在
        if (!options.headers) {
            options.headers = {};
        }

        // 添加授权头部
        const token = localStorage.getItem('backupSystemAuth');
        if (token) {
            options.headers['Authorization'] = 'Bearer ' + token;
        }

        // 记录请求开始时间
        const startTime = Date.now();

        // 发送请求
        const response = await fetch(url, options);

        // 请求结束时间
        const endTime = Date.now();

        // 检查是否未授权
        if (response.status === 401) {
            console.log('会话已过期或未授权，状态码:', response.status);

            // 标记登录状态为过期
            localStorage.removeItem('backupSystemAuth');

            // 隐藏加载动画
            if (showLoadingIndicator) {
                hideLoading();
            }

            // 显示会话过期模态框
            showSessionExpiredModal();

            return null;
        }

        // 先获取响应文本，而不是直接解析JSON
        const responseText = await response.text();

        // 如果响应为空，返回默认成功响应
        if (!responseText || responseText.trim() === '') {
            console.warn('API返回空响应');

            // 隐藏加载动画
            if (showLoadingIndicator) {
                hideLoading();
            }

            return {
                code: 200,
                msg: 'success',
                data: null
            };
        }

        try {
            // 尝试直接解析JSON
            const result = JSON.parse(responseText);

            // 请求成功后确保加载动画按照最小时间显示
            if (showLoadingIndicator) {
                hideLoading();
            }

            // 检查是否在响应体中包含401错误
            if (result.code === 401) {
                console.log('API返回未授权状态:', result.msg);
                localStorage.removeItem('backupSystemAuth');
                showSessionExpiredModal();
                return null;
            }

            return result;
        } catch (jsonError) {
            console.error('JSON解析错误:', jsonError);

            // 尝试清理响应文本
            let cleanText = responseText.trim();

            // 检查并移除可能的BOM标记
            if (cleanText.charCodeAt(0) === 0xFEFF) {
                cleanText = cleanText.substring(1);
            }

            // 尝试处理可能的JSONP响应
            if (cleanText.startsWith('/**') || cleanText.startsWith('/*')) {
                const jsonStart = cleanText.indexOf('{');
                const jsonEnd = cleanText.lastIndexOf('}');
                if (jsonStart !== -1 && jsonEnd !== -1 && jsonEnd > jsonStart) {
                    cleanText = cleanText.substring(jsonStart, jsonEnd + 1);
                }
            }

            // 尝试解析可能被其他字符包围的JSON
            const jsonMatch = cleanText.match(/\{.*\}/s);
            if (jsonMatch) {
                cleanText = jsonMatch[0];
            }

            // 尝试再次解析
            try {
                const result = JSON.parse(cleanText);

                if (showLoadingIndicator) {
                    hideLoading();
                }

                return result;
            } catch (cleanError) {
                console.error('清理后JSON解析仍然失败:', cleanError);

                if (showLoadingIndicator) {
                    hideLoading();
                }

                throw new Error(`服务器返回无效数据: ${jsonError.message}`);
            }
        }
    } catch (error) {
        if (showLoadingIndicator) {
            hideLoading();
        }
        console.error('API请求失败:', error);
        throw error;
    }
}

// 显示会话过期模态框
function showSessionExpiredModal() {
    // 如果已经显示了模态框，不要重复显示
    if (document.querySelector('#session-expired-modal.show')) {
        return;
    }

    // 添加立即登录按钮事件
    document.getElementById('btn-relogin').addEventListener('click', function () {
        window.location.href = '/login.html';
    });

    // 显示模态框
    showModal('session-expired-modal');

    // 5秒后自动跳转到登录页
    setTimeout(() => {
        window.location.href = '/login.html';
    }, 10000);
}

// 下载记录
function downloadRecord(id) {
    const token = localStorage.getItem('backupSystemAuth');
    if (!token) {
        showToast('请先登录', 'warning');
        return;
    }

    // 先检查记录状态
    apiRequest(`/api/records/get?id=${id}`, {}, false, '')
        .then(result => {
            if (!result) return;

            if (result.code === 200) {
                const record = result.data;
                
                // 检查是否被清理
                if (record.status === 'cleaned') {
                    Swal.fire({
                        title: '文件已被清理',
                        text: '此备份文件已被系统清理，无法下载。',
                        icon: 'warning',
                        confirmButtonText: '确定'
                    });
                    return;
                }
                
                // 检查文件路径是否存在
                if (!record.filePath) {
                    Swal.fire({
                        title: '无法下载',
                        text: '该记录没有关联的备份文件。',
                        icon: 'warning',
                        confirmButtonText: '确定'
                    });
                    return;
                }
                
                // 显示下载中提示
                const downloadToast = Swal.fire({
                    title: '正在准备下载...',
                    text: '请稍候',
                    icon: 'info',
                    showConfirmButton: false,
                    allowOutsideClick: false,
                    didOpen: () => {
                        Swal.showLoading();
                    }
                });

                console.log('开始下载记录，ID:', id);

                // 直接使用location.href下载，添加token参数
                window.location.href = `/api/records/download?id=${id}&token=${encodeURIComponent(token)}`;

                // 延迟关闭下载提示，给浏览器一些时间开始下载
                setTimeout(() => {
                    downloadToast.close();
                    showToast('下载已开始，请查看浏览器下载栏', 'success');
                }, 1000);
            } else {
                showToast('获取记录信息失败: ' + result.msg, 'danger');
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showToast('获取记录信息失败: ' + error.message, 'danger');
        });
}

// 删除备份记录
function deleteRecord(id) {
    // 确认对话框
    Swal.fire({
        title: '确认删除',
        text: '确定要删除此备份记录吗？删除后将无法恢复！',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        confirmButtonColor: '#d33',
        cancelButtonColor: '#3085d6'
    }).then((result) => {
        if (result.isConfirmed) {
            // 找到当前行并添加淡出效果
            const recordRow = document.querySelector(`tr:has(button.btn-delete-record[data-id="${id}"])`);
            if (recordRow) {
                recordRow.style.transition = 'opacity 0.5s ease';
                recordRow.style.opacity = '0.5';
            }

            apiRequest(`/api/records/delete?id=${id}`, {
                method: 'POST'
            }, false) // 不使用全局加载动画
                .then(result => {
                    if (result.code === 200) {
                        // 记录被成功删除，可以在UI中立即反映
                        if (recordRow) {
                            recordRow.style.opacity = '0';
                            setTimeout(() => {
                                if (recordRow.parentNode) {
                                    recordRow.parentNode.removeChild(recordRow);
                                }
                            }, 300);
                        }

                        showToast('删除成功', 'success');

                        // 稍微延迟后重新加载记录列表，确保用户可以看到成功消息
                        loadRecords(currentPage, currentPageSize, currentTaskId, false); // 重新加载记录列表，不显示全局加载动画
                    } else {
                        // 恢复行的显示
                        if (recordRow) {
                            recordRow.style.opacity = '1';
                        }

                        showToast(`删除失败: ${result.msg}`, 'danger');
                    }
                })
                .catch(error => {
                    // 恢复行的显示
                    if (recordRow) {
                        recordRow.style.opacity = '1';
                    }

                    console.error('Error:', error);
                    showToast(`删除失败: ${error.message}`, 'danger');
                })
                .finally(() => {
                    // 隐藏区域加载
                    hideSectionLoading('records-panel');
                });
        }
    });
}

// 显示全局加载动画
function showLoading(text = '加载中...') {
    const loadingOverlay = document.getElementById('loading-overlay');
    const loadingText = document.getElementById('loading-text');
    const loadingContent = document.querySelector('.loading-content');

    // 记录显示时间
    window.loadingStartTime = Date.now();

    if (loadingText) {
        loadingText.textContent = text;
    }

    if (loadingOverlay) {
        // 重置加载动画样式
        loadingOverlay.style.opacity = '1';
        if (loadingContent) {
            loadingContent.style.transform = 'scale(1)';
        }

        // 显示加载动画
        loadingOverlay.style.display = 'flex';
    }
}

// 隐藏全局加载动画
function hideLoading() {
    const loadingOverlay = document.getElementById('loading-overlay');
    const loadingContent = document.querySelector('.loading-content');

    // 如果没有加载动画元素，直接返回
    if (!loadingOverlay) return;

    // 计算已经显示的时间
    const now = Date.now();
    const loadingStartTime = window.loadingStartTime || now;
    const loadingDuration = now - loadingStartTime;

    // 最小显示时间 (毫秒)
    const minDuration = 50;

    // 如果显示时间不够，延迟关闭
    if (loadingDuration < minDuration) {
        setTimeout(() => {
            fadeOutLoading(loadingOverlay, loadingContent);
        }, minDuration - loadingDuration);
    } else {
        // 已经显示足够长时间，直接淡出
        fadeOutLoading(loadingOverlay, loadingContent);
    }
}

// 渐变隐藏加载动画
function fadeOutLoading(overlay, content) {
    // 先缩小内容
    if (content) {
        content.style.transform = 'scale(0.8)';
    }

    // 等待一小段时间后开始淡出
    setTimeout(() => {
        // 设置过渡效果
        overlay.style.opacity = '0';

        // 过渡完成后隐藏元素
        setTimeout(() => {
            overlay.style.display = 'none';
            // 重置样式以备下次使用
            overlay.style.opacity = '1';
            if (content) {
                content.style.transform = 'scale(1)';
            }
        }, 400); // 与CSS中的过渡时间一致
    }, 100);
}

// 显示特定区域的加载状态
function showSectionLoading(elementId) {
    const element = document.getElementById(elementId);
    if (element) {
        // 记录该区域的加载开始时间
        window[`${elementId}LoadingStartTime`] = Date.now();

        // 添加加载状态类
        element.classList.add('section-loading');

        // 为该区域添加一个加载指示器
        if (!element.querySelector('.section-loading-indicator')) {
            const indicator = document.createElement('div');
            indicator.className = 'section-loading-indicator';
            indicator.innerHTML = '<div class="section-loading-spinner"></div>';
            element.appendChild(indicator);

            // 渐变显示加载指示器
            setTimeout(() => {
                indicator.style.opacity = '1';
            }, 10);
        }
    }
}

// 隐藏特定区域的加载状态
function hideSectionLoading(elementId) {
    const element = document.getElementById(elementId);
    if (!element) return;

    // 计算已经显示的时间
    const now = Date.now();
    const loadingStartTime = window[`${elementId}LoadingStartTime`] || now;
    const loadingDuration = now - loadingStartTime;

    // 最小显示时间 (毫秒)
    const minDuration = 500;

    // 如果显示时间不够，延迟关闭
    if (loadingDuration < minDuration) {
        setTimeout(() => {
            fadeSectionLoading(element);
        }, minDuration - loadingDuration);
    } else {
        // 已经显示足够长时间，直接淡出
        fadeSectionLoading(element);
    }
}

// 渐变隐藏区域加载状态
function fadeSectionLoading(element) {
    // 获取加载指示器
    const indicator = element.querySelector('.section-loading-indicator');

    if (indicator) {
        // 设置淡出效果
        indicator.style.opacity = '0';

        // 等待过渡完成后移除元素
        setTimeout(() => {
            element.classList.remove('section-loading');
            if (indicator.parentNode) {
                indicator.parentNode.removeChild(indicator);
            }
        }, 300); // 与CSS过渡时间一致
    } else {
        element.classList.remove('section-loading');
    }
}

// 关闭模态框辅助函数
function closeModal(modalId) {
    try {
        console.log(`尝试关闭模态框: ${modalId}`);

        // 方法1: 使用Bootstrap API
        const modalElement = document.getElementById(modalId);
        if (modalElement) {
            const modalInstance = bootstrap.Modal.getInstance(modalElement);
            if (modalInstance) {
                modalInstance.hide();
                console.log(`使用Bootstrap API成功关闭模态框: ${modalId}`);
                return true;
            }
        }

        // 方法2: 尝试使用Bootstrap构造函数
        try {
            const modal = new bootstrap.Modal(document.getElementById(modalId));
            modal.hide();
            console.log(`使用Bootstrap构造函数成功关闭模态框: ${modalId}`);
            return true;
        } catch (e) {
            console.warn(`使用Bootstrap构造函数关闭模态框失败: ${e.message}`);
        }

        // 方法3: 尝试使用jQuery (如果可用)
        if (window.jQuery) {
            $(`#${modalId}`).modal('hide');
            console.log(`使用jQuery成功关闭模态框: ${modalId}`);
            return true;
        }

        // 方法4: 手动修改DOM
        if (modalElement) {
            modalElement.classList.remove('show');
            modalElement.style.display = 'none';
            document.body.classList.remove('modal-open');
            const backdrop = document.querySelector('.modal-backdrop');
            if (backdrop) {
                backdrop.parentNode.removeChild(backdrop);
            }
            console.log(`使用DOM操作成功关闭模态框: ${modalId}`);
            return true;
        }

        console.error(`所有方法都无法关闭模态框: ${modalId}`);
        return false;
    } catch (error) {
        console.error(`关闭模态框时发生错误: ${error.message}`);
        return false;
    }
}

// 显示模态框辅助函数
function showModal(modalId) {
    try {
        const modalElement = document.getElementById(modalId);
        if (!modalElement) {
            console.error(`找不到模态框元素: ${modalId}`);
            return false;
        }

        // 检查是否已存在实例，如果存在则直接使用
        let modalInstance = bootstrap.Modal.getInstance(modalElement);
        
        // 如果不存在实例，创建新实例
        if (!modalInstance) {
            modalInstance = new bootstrap.Modal(modalElement);
        }
        
        // 显示模态框
        modalInstance.show();
        return true;
    } catch (error) {
        console.error(`显示模态框时发生错误: ${error.message}`);
        return false;
    }
}

// 验证Cron表达式
function validateCronExpression(cron) {
    try {
        // 使用cron-validator库验证
        const isValid = cronValidator.isValidCron(cron, {
            seconds: true,  // 支持6位格式
            allowBlankDay: true,  // 允许?通配符
            alias: true,  // 允许别名
            allowSevenAsSunday: true  // 允许7表示周日
        });

        return {
            valid: isValid,
            message: isValid ? 'Cron表达式有效' : '无效的Cron表达式'
        };
    } catch (error) {
        return {
            valid: false,
            message: '无效的Cron表达式'
        };
    }
}

// 验证Cron表达式输入框
function validateCronInput() {
    const cronInput = document.getElementById('task-schedule');
    const cronExpression = cronInput.value.trim();
    const cronFeedback = document.getElementById('cron-feedback');

    // 验证Cron表达式
    const result = validateCronExpression(cronExpression);

    if (result.valid) {
        // 显示有效反馈
        cronInput.classList.remove('is-invalid');
        cronInput.classList.add('is-valid');
        cronFeedback.classList.remove('invalid-feedback');
        cronFeedback.classList.add('valid-feedback');
        cronFeedback.textContent = result.message;
    } else {
        // 显示无效反馈
        cronInput.classList.remove('is-valid');
        cronInput.classList.add('is-invalid');
        cronFeedback.classList.remove('valid-feedback');
        cronFeedback.classList.add('invalid-feedback');
        cronFeedback.textContent = result.message;
    }
}

// 绑定手动执行清理按钮
function bindManualCleanupButton() {
    const btnManualCleanup = document.getElementById('btn-manual-cleanup');
    if (btnManualCleanup) {
        btnManualCleanup.addEventListener('click', executeManualCleanup);
    }
}

// 执行手动清理
function executeManualCleanup() {
    // 获取当前设置的清理天数
    const cleanupDays = document.getElementById('auto-cleanup-days').value;
    
    if (!cleanupDays || cleanupDays <= 0) {
        Swal.fire({
            title: '清理天数无效',
            text: '请设置大于0的清理天数，表示清理多少天前的备份',
            icon: 'warning',
            confirmButtonText: '确定'
        });
        return;
    }
    
    // 确认对话框
    Swal.fire({
        title: '确认执行清理?',
        text: `将清理${cleanupDays}天前的所有备份文件，此操作不可恢复!`,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#d33',
        cancelButtonColor: '#3085d6',
        confirmButtonText: '是，立即清理',
        cancelButtonText: '取消'
    }).then((result) => {
        if (result.isConfirmed) {
            // 用户确认，执行清理
            showLoading('正在执行清理，请稍候...');
            
            // 调用后端API
            apiRequest('/api/cleanup/execute', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            })
                .then(result => {
                    hideLoading();
                    
                    if (result && result.code === 200) {
                        Swal.fire({
                            title: '清理完成',
                            text: result.msg || '清理操作已完成',
                            icon: 'success',
                            confirmButtonText: '确定'
                        });
                    } else {
                        Swal.fire({
                            title: '清理失败',
                            text: (result && result.msg) ? result.msg : '执行清理过程中发生错误',
                            icon: 'error',
                            confirmButtonText: '确定'
                        });
                    }
                })
                .catch(error => {
                    hideLoading();
                    console.error('清理执行失败:', error);
                    
                    Swal.fire({
                        title: '清理失败',
                        text: `执行过程中发生错误: ${error.message}`,
                        icon: 'error',
                        confirmButtonText: '确定'
                    });
                });
        }
    });
}

// 在浏览器后退按钮被点击时重新加载页面状态
window.addEventListener('popstate', function(event) {
    showInitialPanel();
});

// 获取并更新站点信息
function getSiteInfo() {
    // 匿名获取站点信息的API
    fetch('/api/info')
        .then(response => response.json())
        .then(result => {
            if (result.code === 200) {
                // 更新站点名称和版本号
                updateSiteDisplay(result.data.siteName, result.data.version);
            } else {
                console.error('获取站点信息失败:', result.msg);
            }
        })
        .catch(error => {
            console.error('获取站点信息时出错:', error);
        });
}

// 更新站点展示信息
function updateSiteDisplay(siteName, version) {
    // 设置当前年份
    const currentYearElem = document.getElementById('current-year');
    if (currentYearElem) {
        currentYearElem.textContent = new Date().getFullYear();
    }
    
    // 更新站点名称
    if (siteName) {
        const siteNameDisplay = document.getElementById('site-name-display');
        if (siteNameDisplay) {
            siteNameDisplay.textContent = siteName;
        }
        
        // 更新页面标题
        const pageTitle = document.getElementById('page-title');
        if (pageTitle) {
            pageTitle.textContent = siteName;
        }
        
        // 更新页脚
        const footerSiteName = document.getElementById('footer-site-name');
        if (footerSiteName) {
            footerSiteName.textContent = siteName;
        }
    }
    
    // 更新版本号
    if (version) {
        const versionDisplay = document.getElementById('version-display');
        if (versionDisplay) {
            versionDisplay.textContent = `版本: ${version}`;
        }
    }
}