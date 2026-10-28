/**
 * The shape every locale file must satisfy.
 *
 * This is the contract that keeps `en-US.ts` and `zh-CN.ts` in lockstep — adding a key
 * to one locale without the other is a compile error. Keep it limited to keys the app
 * actually renders: unused entries are dead weight in three files at once.
 */
export interface MessageSchema {
  app: { name: string; title: string; titleWithPage: string }
  common: {
    action: {
      back: string
      cancel: string
      edit: string
      save: string
    }
    field: {
      email: string
      password: string
      username: string
      usernameOrEmail: string
      confirmPassword: string
    }
    feedback: {
      networkError: string
    }
    state: {
      disabled: string
      loading: string
      neverUsed: string
    }
    role: { owner: string; user: string }
  }
  apiError: {
    unknown: string
    invalidCredentials: string
    accountDisabled: string
    unauthorized: string
    usernameRequired: string
    usernameTaken: string
    emailTaken: string
    currentPasswordInvalid: string
    passwordTooShort: string
    registrationDisabled: string
    setupRequired: string
    setupCompleted: string
    avatarRequired: string
    avatarInvalidType: string
    avatarTooLarge: string
    profileUpdateFailed: string
    avatarUploadFailed: string
    passwordUpdateFailed: string
    accountDeleteFailed: string
    accountDeleteForbidden: string
    systemLogStreamFailed: string
  }
  route: {
    systemConfig: string
    adminUsers: string
    dashboard: string
    login: string
    notFound: string
    settings: string
    register: string
    setup: string
    systemLogs: string
    tasks: string
    feedback: { loadFailed: string }
  }
  systemConfig: {
    title: string
    description: string
    cardTitle: string
    cardDescription: string
    fields: {
      publicRegistrationEnabled: { title: string; description: string }
      selfServiceAccountDeletionEnabled: { title: string; description: string }
    }
    actions: { retry: string }
    feedback: {
      loadFailedTitle: string
      loadFailed: string
      saving: string
      saved: string
      saveFailed: string
    }
  }
  settings: {
    title: string
    description: string
    tabs: { basic: string; account: string }
    account: {
      profile: string
      profileDesc: string
      profileUpdated: string
      saveProfile: string
      savingProfile: string
      changeAvatar: string
      avatarHint: string
      avatarUnsupportedType: string
      avatarFileTooLarge: string
      avatarProcessFailed: string
      username: string
      usernameRequired: string
      usernamePlaceholder: string
      email: string
      emailNotSet: string
      emailPlaceholder: string
      edit: string
      editProfile: string
      editProfileDesc: string
      statusActive: string
      password: string
      passwordDesc: string
      currentPassword: string
      currentPasswordPlaceholder: string
      newPassword: string
      newPasswordPlaceholder: string
      confirmPassword: string
      confirmPasswordPlaceholder: string
      passwordMismatch: string
      passwordUpdated: string
      updatingPassword: string
      updatePassword: string
      dangerZone: string
      dangerZoneDesc: string
      dangerZoneConfirm: string
      deleteAccount: string
      deleteAccountConfirm: string
      deleteAccountSuccess: string
    }
    basic: {
      theme: string
      themeDesc: string
      colorTheme: string
      light: string
      dark: string
      system: string
      language: string
      selectLanguage: string
      dataRefreshInterval: string
      dataRefreshIntervalDesc: string
    }
  }
  setup: {
    title: string
    creating: string
    success: string
    failed: string
    submit: string
    step1: string
    step2: string
    passwordMismatch: string
    complete: string
  }
  auth: {
    signIn: {
      description: string
      identifierPlaceholder: string
      loginFailed: string
      loginSuccess: string
      noAccount: string
      passwordPlaceholder: string
      register: string
      signingIn: string
      submit: string
      title: string
    }
    signUp: {
      title: string
      description: string
      disabledTitle: string
      disabledDescription: string
      emailOptional: string
      signIn: string
      usernamePlaceholder: string
      emailPlaceholder: string
      passwordPlaceholder: string
      confirmPassword: string
      confirmPasswordPlaceholder: string
      passwordMismatch: string
      creating: string
      registerSuccess: string
      registerFailed: string
      loadFailedTitle: string
      policyLoadFailed: string
      retry: string
      submit: string
    }
  }
  adminUsers: {
    title: string
    description: string
    actions: {
      createUser: string
      retry: string
      disable: string
      enable: string
      previousPage: string
      nextPage: string
    }
    filters: {
      searchPlaceholder: string
      statusPlaceholder: string
      statusAll: string
    }
    table: {
      user: string
      email: string
      role: string
      status: string
      lastActiveAt: string
      createdAt: string
      actions: string
      empty: string
      noEmail: string
      ownerReadonly: string
      pageSummary: string
    }
    status: { active: string; disabled: string }
    dialog: {
      createTitle: string
      createDescription: string
      createSubmit: string
      editTitle: string
      editDescription: string
      editSubmit: string
      usernamePlaceholder: string
      emailPlaceholder: string
      passwordPlaceholder: string
    }
    confirm: {
      disableTitle: string
      disableDescription: string
      enableTitle: string
      enableDescription: string
    }
    feedback: {
      loadFailedTitle: string
      loadFailed: string
      creating: string
      createSuccess: string
      createFailed: string
      updating: string
      updateSuccess: string
      updateFailed: string
      disabling: string
      disableSuccess: string
      disableFailed: string
      enabling: string
      enableSuccess: string
      enableFailed: string
    }
  }
  /** Shared data-table chrome (column toggle + pagination). */
  table: {
    action: { view: string }
    pagination: {
      rowsPerPage: string
      pageOf: string
      goToFirstPage: string
      goToPreviousPage: string
      goToNextPage: string
      goToLastPage: string
      rowSelected: string
    }
  }
  /** Demo page shipped with the starter; see TasksView.vue. */
  tasks: {
    title: string
    description: string
    filterPlaceholder: string
    selectAllAriaLabel: string
    selectRowAriaLabel: string
    openMenuAriaLabel: string
    column: {
      task: string
      title: string
      status: string
      priority: string
    }
    action: { viewTask: string; copyId: string; archive: string }
    filter: { status: string; priority: string }
    label: { documentation: string; bug: string; feature: string }
    status: {
      backlog: string
      todo: string
      inProgress: string
      done: string
      canceled: string
    }
    priority: { low: string; medium: string; high: string }
  }
  home: {
    announcement: string
    title: string
    description: string
    getStarted: string
    viewComponents: string
  }
  dashboard: {
    metric: {
      totalRevenue: { title: string; summary: string; detail: string }
      newCustomers: { title: string; summary: string; detail: string }
      activeAccounts: { title: string; summary: string; detail: string }
      growthRate: { title: string; summary: string; detail: string }
    }
    reservedPanel: {
      eyebrow: string
      title: string
      badge: string
      emptyTitle: string
      emptyDescription: string
      suggestedWidth: string
      suggestedWidthValue: string
      suggestedContent: string
      suggestedContentValue: string
      viewportRule: string
      viewportRuleValue: string
    }
  }
  notFound: { description: string }
  theme: { light: string; dark: string; system: string }
  systemLogs: {
    title: string
    summary: { buffered: string }
    console: { title: string }
    actions: {
      clear: string
      export: string
      pauseFollow: string
      resumeFollow: string
      reconnect: string
    }
    export: {
      title: string
      description: string
      fields: { format: string }
      formats: { csv: string; txt: string; json: string }
      preview: string
    }
    filters: {
      searchPlaceholder: string
      historyPlaceholder: string
      level: {
        allSelected: string
        noneSelected: string
        DEBUG: string
        INFO: string
        WARN: string
        ERROR: string
      }
      history: {
        ALL: string
        100: string
        200: string
        500: string
        1000: string
      }
    }
    empty: { title: string; description: string }
    feedback: { exportSuccess: string; exportEmpty: string }
  }
  nav: {
    main: {
      dashboard: string
      tasks: string
      lifecycle: string
      analytics: string
      projects: string
    }
    management: {
      systemConfig: string
      users: string
      systemLogs: string
      label: string
    }
    secondary: { settings: string; getHelp: string; search: string }
    user: {
      account: string
      billing: string
      notifications: string
      language: string
      logout: string
    }
  }
}
