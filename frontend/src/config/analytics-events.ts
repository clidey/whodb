/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export const ANALYTICS_EVENTS = {
    DESKTOP_APP_LAUNCHED: 'desktop_app_launched',

    UI_SCREEN_VIEWED: 'ui.screen_viewed',
    UI_FORM_OPENED: 'ui.form_opened',
    UI_FORM_SUBMITTED: 'ui.form_submitted',
    UI_FORM_ABANDONED: 'ui.form_abandoned',
    UI_OPTION_CHANGED: 'ui.option_changed',
    UI_SETTINGS_VIEWED: 'ui.settings_viewed',
    UI_TELEMETRY_TOGGLED: 'ui.telemetry_toggled',
    UI_STORAGE_UNIT_VIEWED: 'ui.storage_unit_viewed',
    UI_STORAGE_UNIT_CREATE_TOGGLE: 'ui.storage_unit_create_toggle',
    UI_STORAGE_UNIT_FIELD_ADDED: 'ui.storage_unit_field_added',
    UI_STORAGE_UNIT_CREATED: 'ui.storage_unit_created',
    UI_STORAGE_UNIT_CREATE_BLOCKED: 'ui.storage_unit_create_blocked',
    UI_STORAGE_UNIT_CREATE_FAILED: 'ui.storage_unit_create_failed',

    AUTH_LOGIN_BLOCKED: 'auth.login_blocked',
    AUTH_LOGIN_FAILED: 'auth.login_failed',
    AUTH_LOGIN_SUCCEEDED: 'auth.login_succeeded',
    AUTH_CONNECTION_TEST_SUBMITTED: 'auth.connection_test_submitted',
    AUTH_CONNECTION_TEST_SUCCEEDED: 'auth.connection_test_succeeded',
    AUTH_CONNECTION_TEST_FAILED: 'auth.connection_test_failed',
    AUTH_CLOUD_CONNECTION_PREFILLED: 'auth.cloud_connection_prefilled',
    AUTH_DATABASE_FILE_PICKER_OPENED: 'auth.database_file_picker_opened',
    AUTH_DATABASE_FILE_SELECTED: 'auth.database_file_selected',

    CHAT_MESSAGE_DRAFT_ABANDONED: 'chat.message_draft_abandoned',
    CHAT_MESSAGE_SUBMITTED: 'chat.message_submitted',
    CHAT_MESSAGE_COMPLETED: 'chat.message_completed',
    CHAT_MESSAGE_FAILED: 'chat.message_failed',
    CHAT_EXAMPLE_SELECTED: 'chat.example_selected',
    CHAT_CLEARED: 'chat.cleared',
    CHAT_SQL_CONFIRMATION_ACCEPTED: 'chat.sql_confirmation_accepted',
    CHAT_SQL_CONFIRMATION_DECLINED: 'chat.sql_confirmation_declined',
    CHAT_SQL_CONFIRMATION_QUERY_TOGGLED: 'chat.sql_confirmation_query_toggled',
    CHAT_SESSION_CREATED: 'chat.session_created',
    CHAT_SESSION_SELECTED: 'chat.session_selected',
    CHAT_SESSION_DELETED: 'chat.session_deleted',
    CHAT_SESSION_RENAMED: 'chat.session_renamed',

    CLOUD_PROVIDER_CREATE_OPENED: 'cloud_provider.create_opened',
    CLOUD_PROVIDER_EDIT_OPENED: 'cloud_provider.edit_opened',
    CLOUD_PROVIDER_DELETE_CLICKED: 'cloud_provider.delete_clicked',
    CLOUD_PROVIDER_DELETED: 'cloud_provider.deleted',
    CLOUD_PROVIDER_DELETE_FAILED: 'cloud_provider.delete_failed',
    CLOUD_PROVIDER_REFRESH_CLICKED: 'cloud_provider.refresh_clicked',
    CLOUD_PROVIDER_REFRESHED: 'cloud_provider.refreshed',
    CLOUD_PROVIDER_REFRESH_FAILED: 'cloud_provider.refresh_failed',
    CLOUD_PROVIDER_LOCAL_PROFILE_SELECTED: 'cloud_provider.local_profile_selected',
    CLOUD_PROVIDER_LOCAL_PROJECT_SELECTED: 'cloud_provider.local_project_selected',
    CLOUD_PROVIDER_SUBSCRIPTION_SELECTED: 'cloud_provider.subscription_selected',
    CLOUD_PROVIDER_SAVE_BLOCKED: 'cloud_provider.save_blocked',
    CLOUD_PROVIDER_SAVED: 'cloud_provider.saved',
    CLOUD_PROVIDER_SAVE_FAILED: 'cloud_provider.save_failed',
    CLOUD_PROVIDER_TEST_CLICKED: 'cloud_provider.test_clicked',
    CLOUD_PROVIDER_TEST_BLOCKED: 'cloud_provider.test_blocked',
    CLOUD_PROVIDER_TEST_SUCCEEDED: 'cloud_provider.test_succeeded',
    CLOUD_PROVIDER_TEST_FAILED: 'cloud_provider.test_failed',
} as const;

export type AnalyticsEventName = typeof ANALYTICS_EVENTS[keyof typeof ANALYTICS_EVENTS];

export const SAFE_ANALYTICS_PROPERTY_KEYS = new Set([
    'action',
    'app_type',
    'auth_mode',
    'auto_scroll_enabled',
    'billing_interval',
    'build_edition',
    'build_environment',
    'connection_mode',
    'database_type',
    'decision',
    'deployment',
    'direction',
    'enabled',
    'error_code',
    'field',
    'form',
    'group_type',
    'has_advanced_fields',
    'has_custom_slug',
    'has_database',
    'has_model',
    'has_password',
    'has_profile',
    'has_project',
    'has_provider',
    'has_source',
    'has_token',
    'input_method',
    'interval',
    'is_desktop',
    'is_embedded',
    'is_first_login',
    'is_super_admin',
    'is_template',
    'language',
    'mode',
    'model_type',
    'node_type',
    'open',
    'operation_type',
    'outcome',
    'partner_cohort',
    'plan',
    'platform',
    'provider_type',
    'route',
    'scope',
    'section',
    'selected',
    'source',
    'status',
    'step',
    'tab',
    'trigger',
    'view_mode',
]);

export const SAFE_ANALYTICS_PROPERTY_SUFFIXES = [
    '_bucket',
    '_count',
    '_enabled',
    '_index',
    '_mode',
    '_present',
    '_selected',
    '_type',
    '_visible',
];
