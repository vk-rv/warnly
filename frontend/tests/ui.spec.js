import { test, expect } from '@playwright/test';

const user = { ID: 1, Username: 'admin', Email: 'admin@example.com' };
test.beforeEach(async ({ page }) => {
  await page.route('**/api/session', (route) => route.fulfill({ json: user }));
});

test('login failure and successful session', async ({ page }) => {
  await page.route('**/api/login', async (route) => {
    if (route.request().method() === 'GET') return route.fulfill({ json: { IsDemo: true } });
    const body = new URLSearchParams(route.request().postData());
    return route.fulfill(
      body.get('password') === 'correct'
        ? { json: user }
        : { status: 401, json: { error: 'Invalid login credentials.' } }
    );
  });
  await page.route('**/api/issues', (route) =>
    route.fulfill({ json: { Issues: [], Projects: [], TotalIssues: 0 } })
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
    route.fulfill({ json: [{ ID: 1, Name: 'Core' }] })
  );
  await page.route('**/api/projects', async (route) => {
    expect(new URLSearchParams(route.request().postData()).get('platform')).toBe('rust');
    return route.fulfill({
      json: { ID: 7, Name: 'Service', DSN: 'http://key@localhost/ingest/7', Platform: 'rust' }
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
        IssueID: 2,
        ErrorType: '<script>alert(1)</script>',
        Message: 'Failure',
        TimesSeen: 4,
        UserCount: 1,
        Teammates: [{ ID: 1, Username: 'admin' }],
        Assignments: { IssueToAssigned: {} },
        StackDetails: []
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
    return route.fulfill({ json: { Messages: messages, Teammates: [] } });
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
    route.fulfill({ json: { Projects: [{ ID: 7, Name: 'Service' }] } })
  );
  let body;
  await page.route('**/api/alerts', (route) => {
    if (route.request().method() === 'POST') {
      body = new URLSearchParams(route.request().postData());
      return route.fulfill({ status: 204 });
    }
    return route.fulfill({ json: { Alerts: [], Projects: [], TotalAlerts: 0 } });
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

test('expired sessions redirect to login', async ({ page }) => {
  await page.route('**/api/session', (route) =>
    route.fulfill({ status: 401, json: { error: 'Authentication required' } })
  );
  await page.route('**/api/login', (route) => route.fulfill({ json: {} }));
  await page.goto('/projects');
  await expect(page).toHaveURL('/login');
  await expect(page.getByLabel('Email or username')).toBeVisible();
});

test('project issues retain their project in links', async ({ page }) => {
  await page.route('**/api/projects/7', (route) =>
    route.fulfill({
      json: {
        Project: {
          ID: 7,
          Name: 'Service',
          AllLength: 1,
          ResultIssueList: [
            {
              ID: 3,
              ProjectID: 0,
              Type: 'Failure',
              Message: 'Error',
              LastSeen: '2026-01-01T00:00:00Z'
            }
          ]
        },
        Period: '24h',
        Issues: 'all'
      }
    })
  );
  await page.goto('/projects/7');
  await expect(page.getByRole('link', { name: 'Failure' })).toHaveAttribute(
    'href',
    '/projects/7/issues/3?source=issue'
  );
  await expect(page.getByRole('button', { name: 'Period: Last 24 hours', exact: true })).toBeVisible();
});
