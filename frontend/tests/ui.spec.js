import { test, expect } from '@playwright/test';

const user = { id: 1, username: 'admin', email: 'admin@example.com' };
test.beforeEach(async ({ page }) => {
  await page.route('**/api/session', (route) => route.fulfill({ json: user }));
});

test('login failure and successful session', async ({ page }) => {
  await page.route('**/api/login', async (route) => {
    if (route.request().method() === 'GET') return route.fulfill({ json: { is_demo: true } });
    const body = new URLSearchParams(route.request().postData());
    return route.fulfill(
      body.get('password') === 'correct'
        ? { json: user }
        : { status: 401, json: { error: 'Invalid login credentials.' } }
    );
  });
  await page.route('**/api/issues', (route) =>
    route.fulfill({ json: { issues: [], projects: [], total_issues: 0 } })
  );
  await page.goto('/login');
  await page.getByLabel('Email or username').fill('admin');
  await page.getByLabel('Password', { exact: true }).fill('wrong');
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.getByRole('alert')).toHaveText('Invalid login credentials.');
  await page.getByLabel('Password', { exact: true }).fill('correct');
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Issues', exact: true })).toBeVisible();
});

test('create a project and show its DSN', async ({ page }) => {
  await page.route('**/api/projects/new', (route) =>
    route.fulfill({ json: [{ id: 1, name: 'Core' }] })
  );
  await page.route('**/api/projects', async (route) => {
    expect(new URLSearchParams(route.request().postData()).get('platform')).toBe('rust');
    return route.fulfill({
      json: { id: 7, name: 'Service', dsn: 'http://key@localhost/ingest/7', platform: 'rust' }
    });
  });
  await page.goto('/projects/new');
  await page.getByLabel('Project name').fill('Service');
  await page.getByLabel('Platform').selectOption('rust');
  await page.getByRole('button', { name: 'Create project' }).click();
  await expect(page.getByText('http://key@localhost/ingest/7')).toBeVisible();
});

test('deep issue link, escaped content, assignment and discussion', async ({ page }) => {
  let assignment;
  await page.route('**/api/projects/1/issues/2', (route) =>
    route.fulfill({
      json: {
        issue_id: 2,
        error_type: '<script>alert(1)</script>',
        message: 'Failure',
        times_seen: 4,
        user_count: 1,
        teammates: [{ id: 1, username: 'admin' }],
        assignments: { issue_to_assigned: {} },
        stack_details: []
      }
    })
  );
  await page.route('**/api/projects/1/issues/2/assignments', (route) => {
    assignment = route.request().postData();
    return route.fulfill({ status: 204 });
  });
  let messages = [];
  await page.route('**/api/projects/1/issues/2/discussions', (route) => {
    if (route.request().method() === 'POST')
      messages = [
        {
          id: 1,
          user_id: 1,
          username: 'admin',
          created_at: new Date().toISOString(),
          content: new URLSearchParams(route.request().postData()).get('content')
        }
      ];
    return route.fulfill({ json: { messages: messages, teammates: [] } });
  });
  await page.goto('/projects/1/issues/2');
  await expect(page.getByRole('heading', { name: '<script>alert(1)</script>' })).toBeVisible();
  await page.reload();
  await page.getByLabel('Assigned to').selectOption('1');
  await expect.poll(() => assignment).toBe('user_id=1');
  await page.getByRole('link', { name: 'Discussion', exact: true }).click();
  await page.getByLabel('Comment', { exact: true }).fill('Investigating');
  await page.getByRole('button', { name: 'Post comment' }).click();
  await expect(page.getByText('Investigating', { exact: true })).toBeVisible();
});

test('alert creation sends the rule fields', async ({ page }) => {
  await page.route('**/api/alerts/new', (route) =>
    route.fulfill({ json: { projects: [{ id: 7, name: 'Service' }] } })
  );
  let body;
  await page.route('**/api/alerts', (route) => {
    if (route.request().method() === 'POST') {
      body = new URLSearchParams(route.request().postData());
      return route.fulfill({ status: 204 });
    }
    return route.fulfill({ json: { alerts: [], projects: [], total_alerts: 0 } });
  });
  await page.goto('/alerts/new');
  await page.getByLabel('Rule name').fill('Error spike');
  await page.getByLabel('Threshold').fill('10');
  await page.getByLabel('High priority').check();
  await page.getByRole('button', { name: 'Save rule' }).click();
  await expect(page).toHaveURL('/alerts');
  expect(body.get('project_id')).toBe('7');
  expect(body.get('threshold')).toBe('10');
  expect(body.get('high_priority')).toBe('true');
});

for (const path of ['/', '/projects']) {
  for (const unauthorizedEndpoint of ['/session', path === '/' ? '/issues' : path]) {
    test(`${path} redirects before rendering when ${unauthorizedEndpoint} returns 401`, async ({
      page
    }) => {
      const documents = [];
      page.on('request', (request) => {
        if (request.isNavigationRequest()) documents.push(request.url());
      });
      await page.addInitScript(() => {
        window.protectedContentRendered = false;
        new MutationObserver(() => {
          if (
            document.querySelector('.sidebar, .page-heading') ||
            document.body?.textContent.includes('Authentication required')
          )
            window.protectedContentRendered = true;
        }).observe(document, { childList: true, subtree: true });
      });
      await page.route(`**/api${unauthorizedEndpoint}`, (route) =>
        route.fulfill({ status: 401, json: { error: 'Authentication required' } })
      );
      await page.route('**/api/login', (route) => route.fulfill({ json: {} }));
      await page.goto(path);
      await expect(page).toHaveURL('/login');
      await expect(page.getByLabel('Email or username')).toBeVisible();
      expect(documents).toHaveLength(1);
      expect(await page.evaluate(() => window.protectedContentRendered)).toBe(false);
    });
  }
}

test('project issues retain their project in links', async ({ page }) => {
  await page.route('**/api/projects/7', (route) =>
    route.fulfill({
      json: {
        project: {
          id: 7,
          name: 'Service',
          all_length: 1,
          result_issue_list: [
            {
              id: 3,
              project_id: 0,
              type: 'Failure',
              message: 'Error',
              last_seen: '2026-01-01T00:00:00Z'
            }
          ]
        },
        period: '24h',
        issues: 'all'
      }
    })
  );
  await page.goto('/projects/7');
  await expect(page.getByRole('link', { name: 'Failure' })).toHaveAttribute(
    'href',
    '/projects/7/issues/3?source=issue'
  );
  await expect(
    page.getByRole('button', { name: 'period: Last 24 hours', exact: true })
  ).toBeVisible();
});

test('login errors stay inside the form and retain credentials', async ({ page }) => {
  await page.route('**/api/login', (route) =>
    route.fulfill(
      route.request().method() === 'GET'
        ? { json: {} }
        : { status: 401, json: { error: 'Invalid login credentials.' } }
    )
  );
  await page.goto('/login');
  await page.getByLabel('Email or username').fill('wrong-user');
  await page.getByLabel('Password', { exact: true }).fill('wrong-password');
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.getByRole('alert')).toHaveCount(1);
  await expect(page.locator('.login-fields [role=alert]')).toHaveText('Invalid login credentials.');
  await expect(page.getByLabel('Email or username')).toHaveValue('wrong-user');
  await expect(page.getByLabel('Password', { exact: true })).toHaveAttribute(
    'aria-invalid',
    'true'
  );
  await page.screenshot({ path: 'test-results/login-desktop.png', fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.locator('.login-intro')).toBeHidden();
  await expect(page.getByRole('button', { name: 'Sign in', exact: true })).toBeVisible();
  await page.screenshot({ path: 'test-results/login-mobile.png', fullPage: true });
});

test('OIDC login offers only the configured provider', async ({ page }) => {
  await page.route('**/api/login', (route) =>
    route.fulfill({ json: { auth_url: '/oidc/start', provider_name: 'Keycloak' } })
  );
  await page.goto('/login');
  await expect(page.getByLabel('Password', { exact: true })).toHaveCount(0);
  await expect(page.getByRole('link', { name: 'Sign in with Keycloak' })).toHaveAttribute(
    'href',
    '/oidc/start'
  );
});

test('discussion preview, mentions and failed submission preserve the draft', async ({ page }) => {
  await page.route('**/api/projects/1/issues/2', (route) =>
    route.fulfill({ json: { error_type: 'Error', issue_id: 2 } })
  );
  let posted;
  await page.route('**/api/projects/1/issues/2/discussions', (route) => {
    if (route.request().method() === 'POST') {
      posted = new URLSearchParams(route.request().postData());
      return route.fulfill({ status: 500, json: { error: 'Unable to post comment' } });
    }
    return route.fulfill({
      json: {
        teammates: [{ id: 1, username: 'admin' }],
        messages: [
          {
            id: 1,
            user_id: 1,
            username: 'admin',
            content: 'Investigating\n@admin',
            created_at: new Date().toISOString()
          }
        ]
      }
    });
  });
  await page.goto('/projects/1/issues/2/discussions');
  await page.getByLabel('Comment', { exact: true }).fill('Hello @ad');
  await page.locator('.mention-menu').getByRole('button', { name: 'admin', exact: true }).click();
  await page.getByRole('tab', { name: 'Preview' }).click();
  await expect(page.getByRole('tabpanel').locator('strong')).toHaveText('@admin');
  await page.getByRole('button', { name: 'Post comment' }).click();
  await expect(page.getByRole('alert')).toHaveText('Unable to post comment');
  expect(posted.getAll('mentioned_users')).toEqual(['1']);
  await page.getByRole('tab', { name: 'Write' }).click();
  await expect(page.getByLabel('Comment', { exact: true })).toHaveValue('Hello @admin ');
  await page.screenshot({ path: 'test-results/discussion-desktop.png', fullPage: true });
});

test('system restores sorting, query details and schema columns', async ({ page }) => {
  await page.route('**/api/system', (route) =>
    route.fulfill({
      json: [
        {
          normalized_query: 'SELECT fast FROM events',
          avg_duration: 1,
          total_calls: 3,
          total_read_bytes: '1 KiB',
          percent: 10
        },
        {
          normalized_query: 'SELECT slow FROM events',
          avg_duration: 100,
          total_calls: 12,
          total_read_bytes: '9 KiB',
          percent: 90
        }
      ]
    })
  );
  await page.route('**/api/system/schema', (route) =>
    route.fulfill({
      json: [
        {
          name: 'events',
          readable_bytes: '10 MiB',
          total_rows: 1200,
          engine: 'MergeTree',
          partition_key: 'toDate(timestamp)',
          total_bytes: 10485760
        }
      ]
    })
  );
  await page.goto('/system');
  await expect(page.locator('tbody tr').first()).toContainText('SELECT slow');
  await page.getByRole('button', { name: /Avg duration/ }).click();
  await expect(page.locator('tbody tr').first()).toContainText('SELECT fast');
  await page.locator('.query-detail summary').first().click();
  await expect(page.getByRole('button', { name: 'Copy SQL' })).toBeVisible();
  await page.screenshot({ path: 'test-results/system-desktop.png', fullPage: true });
  await page.getByRole('link', { name: 'Schema', exact: true }).click();
  await expect(page.getByRole('columnheader')).toHaveCount(5);
  await expect(page.getByRole('cell', { name: '1,200', exact: true })).toBeVisible();
});

test('saved notification matches original dark toast and disappears', async ({ page }) => {
  await page.route('**/api/settings', (route) => route.fulfill({ json: { webhook: {} } }));
  await page.route('**/api/settings/webhook', (route) => route.fulfill({ status: 204 }));
  await page.goto('/settings');
  await page.getByRole('button', { name: 'Save webhook' }).click();
  const toast = page.getByRole('status');
  await expect(toast).toHaveText('Saved');
  await expect(toast).toHaveCSS('background-color', 'rgba(35, 39, 47, 0.92)');
  await expect(toast).toBeHidden({ timeout: 5000 });
});

test('project row opens issue from whitespace, errors and users but not responsible', async ({
  page
}) => {
  let assigned = false;
  await page.route('**/api/projects/2?period=90d', (route) =>
    route.fulfill({
      json: {
        project: {
          id: 2,
          name: 'Service',
          all_length: 1,
          result_issue_list: [
            {
              id: 3,
              project_id: 2,
              type: 'Failure',
              message: 'Error',
              times_seen: 42,
              user_count: 7
            }
          ]
        },
        period: '90d',
        issues: 'all',
        teammates: [{ id: 1, username: 'admin' }]
      }
    })
  );
  await page.route('**/api/projects/2/issues/3?source=issue', (route) =>
    route.fulfill({ json: { issue_id: 3, error_type: 'Failure' } })
  );
  await page.route('**/api/projects/2/issues/3/assignments', (route) => {
    assigned = true;
    return route.fulfill({ status: 204 });
  });
  for (const column of [0, 1, 2]) {
    await page.goto('/projects/2?period=90d');
    const cell = page.locator('.project-issue-table tbody tr').first().locator('td').nth(column);
    const box = await cell.boundingBox();
    await page.mouse.click(box.x + box.width - 5, box.y + box.height / 2);
    await expect(page).toHaveURL(/\/projects\/2\/issues\/3\?source=issue$/);
  }
  await page.goto('/projects/2?period=90d');
  const responsible = page.locator('.issue-responsible');
  const box = await responsible.boundingBox();
  await page.mouse.click(box.x + 3, box.y + 3);
  await expect(page).toHaveURL(/\/projects\/2\?period=90d$/);
  await page.getByLabel('Responsible for Failure').selectOption('1');
  await expect.poll(() => assigned).toBe(true);
  await expect(page).toHaveURL(/\/projects\/2\?period=90d$/);
});

test('event opens from every column and row whitespace', async ({ page }) => {
  await page.route('**/api/projects/2/issues/8*', (route) =>
    route.fulfill({ json: { issue_id: 8, error_type: 'Failure' } })
  );
  await page.route('**/api/projects/2/issues/8/events', (route) =>
    route.fulfill({
      json: {
        events: [
          {
            event_id: 'event-1',
            title: 'Failure',
            message: 'Details',
            created_at: '2026-10-14T12:00:00Z',
            env: 'production',
            user_email: 'user@example.com'
          }
        ],
        total_events: 1
      }
    })
  );
  for (const column of [0, 1, 2, 3]) {
    await page.goto('/projects/2/issues/8/events');
    const cell = page.locator('.event-rows tr').first().locator('td').nth(column);
    const box = await cell.boundingBox();
    await page.mouse.click(box.x + box.width - 5, box.y + box.height - 5);
    await expect(page).toHaveURL(/\/projects\/2\/issues\/8\?event_id=event-1$/);
  }
});

test('home issue opens from every column including whitespace', async ({ page }) => {
  await page.route('**/api/issues', (route) =>
    route.fulfill({
      json: {
        projects: [{ id: 2, name: 'Service' }],
        issues: [
          {
            id: 8,
            project_id: 2,
            type: 'Failure',
            message: 'Details',
            times_seen: 42,
            user_count: 7,
            first_seen: '2026-10-14T12:00:00Z',
            last_seen: '2026-10-14T13:00:00Z'
          }
        ],
        total_issues: 1
      }
    })
  );
  await page.route('**/api/projects/2/issues/8?source=issue', (route) =>
    route.fulfill({ json: { issue_id: 8, error_type: 'Failure' } })
  );
  for (const column of [0, 1, 2, 3, 4, 5]) {
    await page.goto('/');
    const cell = page.locator('.issue-table tbody tr').first().locator('td').nth(column);
    const box = await cell.boundingBox();
    await page.mouse.click(box.x + box.width - 5, box.y + box.height - 5);
    await expect(page).toHaveURL(/\/projects\/2\/issues\/8\?source=issue$/);
  }
});

test('issue sidebar restores expandable field percentages from API', async ({ page }) => {
  await page.route('**/api/projects/2/issues/10?source=issue', (route) =>
    route.fulfill({
      json: {
        issue_id: 10,
        error_type: 'Failure',
        last_event: { tags_key: ['environment', 'release'], tags_value: ['production', 'v2'] },
        tag_count: [{ tag: 'environment' }, { tag: 'release' }],
        tag_value_num: [
          { tag: 'environment', value: 'production', percents_of_total: 76.9 },
          { tag: 'environment', value: 'staging', percents_of_total: 23.1 },
          { tag: 'release', value: 'v2', percents_of_total: 100 }
        ]
      }
    })
  );
  await page.goto('/projects/2/issues/10?source=issue');
  const fields = page.getByRole('region', { name: 'Field distributions' });
  const environment = fields.locator('details').first();
  await expect(environment.locator('.field-current')).toHaveText('production');
  await expect(environment.locator('.field-percent').first()).toBeHidden();
  await environment.locator('summary').click();
  await expect(environment.locator('.field-percent')).toHaveText(['76%', '23%']);
  await expect(environment.locator('.field-distribution-bar > span')).toHaveAttribute(
    'style',
    /width: 75%/
  );
  await fields.locator('details').nth(1).locator('summary').click();
  await expect(fields.locator('details').nth(1).locator('.field-percent')).toHaveText('100%');
});

test('fields render grouped cards with issue filter links', async ({ page }) => {
  await page.route('**/api/projects/2/issues/10', (route) =>
    route.fulfill({ json: { issue_id: 10, error_type: 'Failure' } })
  );
  await page.route('**/api/projects/2/issues/10/fields', (route) =>
    route.fulfill({
      json: {
        project_name: 'Order API',
        tag_count: [{ tag: 'environment' }, { tag: 'release' }],
        field_value_num: [
          { tag: 'environment', value: 'production eu', count: 12 },
          { tag: 'release', value: 'v2', count: 5 }
        ]
      }
    })
  );
  await page.goto('/projects/2/issues/10/fields');
  const cards = page.locator('.issue-field-card');
  await expect(cards).toHaveCount(2);
  await expect(cards.first().getByRole('heading')).toHaveText('environment');
  await expect(cards.first().locator('.field-count')).toHaveText('12');
  const href = await cards.first().getByRole('link').getAttribute('href');
  const params = new URL(href, 'http://localhost').searchParams;
  expect(params.get('project_name')).toBe('Order API');
  expect(params.get('query')).toBe('environment:"production eu"');
  expect(params.get('period')).toBe('14d');
  const first = await cards.nth(0).boundingBox();
  const second = await cards.nth(1).boundingBox();
  expect(first.y).toBe(second.y);
  await page.setViewportSize({ width: 390, height: 844 });
  const mobileFirst = await cards.nth(0).boundingBox();
  const mobileSecond = await cards.nth(1).boundingBox();
  expect(mobileSecond.y).toBeGreaterThan(mobileFirst.y + mobileFirst.height);
});

test('delete dialog handles cancel, failure and retry with favicon present', async ({ page }) => {
  let calls = 0;
  await page.route('**/api/alerts', (route) =>
    route.fulfill({
      json: {
        alerts: [{ id: 1, rule_name: 'Production errors', status: 'Active' }],
        projects: [],
        total_alerts: 1
      }
    })
  );
  await page.route('**/api/alerts/1', (route) => {
    calls++;
    return route.fulfill(
      calls === 1 ? { status: 500, json: { error: 'Could not delete alert' } } : { status: 204 }
    );
  });
  await page.goto('/alerts');
  await expect(page.locator('link[rel=icon]')).toHaveAttribute('href', /\.svg|data:image\/svg/);
  const trigger = page.getByRole('button', { name: 'Delete', exact: true });
  await trigger.click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  await expect(dialog).toContainText('Production errors');
  await expect(dialog.getByRole('button', { name: 'Cancel', exact: true })).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(dialog).toHaveCount(0);
  expect(calls).toBe(0);
  await expect(trigger).toBeFocused();
  await trigger.click();
  await page.screenshot({ path: 'test-results/delete-dialog.png', animations: 'disabled' });
  await dialog.getByRole('button', { name: 'Delete alert', exact: true }).click();
  await expect(dialog.getByRole('alert')).toHaveText('Could not delete alert');
  await dialog.getByRole('button', { name: 'Delete alert', exact: true }).click();
  await expect(dialog).toHaveCount(0);
  expect(calls).toBe(2);
});
