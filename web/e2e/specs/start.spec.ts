import { expect, test } from '@playwright/test'
import { CatalogApi, defaultPreferences } from '../support/catalog-api'
import { StartPage } from '../support/start-page'

test.describe('Start product journeys', () => {
  test.beforeEach(async ({ request }) => new CatalogApi(request).resetUserExperience())
  test.afterEach(async ({ request }) => new CatalogApi(request).removeTestChanges())

  test('find and open an internal tool using only the keyboard', async ({ page }) => {
    const start = new StartPage(page)

    await test.step('Open the personalized, categorized launchpad', async () => {
      await start.open()
      await start.expectAdministratorHome()
    })
    await test.step('Find tools by description and URL', async () => {
      await start.searchFor('passkeys')
      await start.expectSingleResult('Pocket ID')
      await start.searchFor('pdf.internal')
      await start.expectResult('Stirling PDF')
      await start.clearSearch()
    })
    await test.step('Use keyboard help, navigation, and view switching', async () => {
      await start.focusSearchWithKeyboard()
      await start.searchFor('not-a-destination')
      await start.clearSearchWithKeyboard()
      await start.openKeyboardHelp()
      await start.closeKeyboardHelp()
      await start.selectNextLinkWithKeyboard('Pocket ID')
      await start.toggleViewWithKeyboard('All links')
      await expect(page.getByRole('heading', { name:'All links' })).toBeVisible()
      await start.toggleViewWithKeyboard('Categories')
    })
    await test.step('Open the only matching destination with Enter', async () => {
      await page.route('https://id.internal.example/**', route => route.fulfill({ body:'<title>Opened by keyboard</title>' }))
      await start.searchFor('Pocket ID')
      await start.openOnlySearchResult()
      await expect(page).toHaveURL('https://id.internal.example/')
    })
  })

  test('customize the launchpad and keep those choices after reload', async ({ page }) => {
    const start = new StartPage(page)
    await start.open()

    await test.step('Open settings and choose a compact hacker theme', async () => {
      const settings = await start.openDisplaySettings()
      await settings.chooseTheme('hacker')
      await settings.showDescriptions(false)
      await settings.showCategoryHeadings(false)
      await settings.showIcons(false)
      await expect(page.locator('html')).toHaveAttribute('data-theme', 'hacker')
      await expect(page.locator('.description')).toHaveCount(0)
      await expect(page.locator('.link-section > header')).toHaveCount(0)
      await expect(page.locator('.link-icon')).toHaveCount(0)
      await settings.close()
    })
    await test.step('Reload and see the same personalized experience', async () => {
      await page.reload()
      await expect(page.locator('html')).toHaveAttribute('data-theme', 'hacker')
      await expect(page.locator('.description')).toHaveCount(0)
      const persisted = await page.evaluate(() => JSON.parse(localStorage.getItem('start.preferences.e2e-admin@example.test') ?? '{}'))
      expect(persisted).toMatchObject({ theme:'hacker', showDescriptions:false })
    })
  })

  test('hide an irrelevant tool and bring it back later', async ({ page }) => {
    const start = new StartPage(page)
    await start.open()

    await test.step('Hide the currently selected Stirling PDF link', async () => {
      await expect(page.locator('.link-card.active')).toContainText('Stirling PDF')
      await start.hideSelectedLinkWithKeyboard()
      await expect(start.linkCard('Stirling PDF')).toHaveCount(0)
    })
    await test.step('Show hidden links in settings and restore Stirling PDF', async () => {
      let settings = await start.openDisplaySettings()
      await settings.showHiddenLinks(true)
      await expect(start.linkCard('Stirling PDF')).toHaveClass(/hidden/)
      await settings.close()
      await start.showHiddenLink('Stirling PDF')
      settings = await start.openDisplaySettings()
      await settings.showHiddenLinks(false)
      await settings.close()
    })
  })

  test('manage a link through its complete lifecycle', async ({ page }) => {
    const suffix = Date.now().toString(36)
    const title = `E2E Toolbox ${suffix}`
    const updatedTitle = `${title} Updated`
    const slug = `e2e-toolbox-${suffix}`
    const start = new StartPage(page)
    await start.open()

    await test.step('Create a link and preview its chosen icon', async () => {
      const editor = await start.openNewLinkEditorWithKeyboard()
      await editor.fillLink({ title, slug, url:'http://toolbox.e2e.test', description:'Created through the browser test', iconProvider:'material', iconName:'rocket_launch', tags:'e2e, toolbox' })
      await editor.expectMaterialIconPreview('rocket_launch')
      await editor.enterAudienceRules('developer', 'platform')
      // The development identity has no role/team mapping, so remove the rules
      // before saving while still proving that the form accepts them.
      await editor.clearAudienceRules()
      await editor.save()
      await expect(start.linkCard(title)).toBeVisible()
    })
    await test.step('Edit the title and see the new icon preview', async () => {
      const editor = await start.openLinkEditor(title)
      await editor.changeTitle(updatedTitle)
      await editor.changeIcon('construction')
      await editor.expectMaterialIconPreview('construction')
      await editor.save()
    })
    await test.step('Restore the first revision of this link', async () => {
      const history = await start.openLinkHistory(updatedTitle)
      await history.expectRevision(1)
      await history.restoreRevision(1)
      await expect(start.linkCard(title)).toBeVisible()
    })
    await test.step('Delete the link, then recover it from administrative history', async () => {
      await start.deleteLink(title)
      const history = await start.openAdministrativeHistory()
      await history.filterFor(slug)
      await history.selectRevision(1)
      await history.restoreSelected(1)
      await history.close()
      await expect(start.linkCard(title)).toBeVisible()
    })
  })

  test('import a reviewed YAML catalog as a system administrator', async ({ page }) => {
    const suffix=Date.now().toString(36)
    const slug=`e2e-import-${suffix}`
    const title=`Imported Runbook ${suffix}`
    const start=new StartPage(page)
    await start.open()

    await test.step('Open the importer from Settings and review a YAML file',async()=>{
      const importer=await start.openCatalogImportFromSettings()
      await importer.importYaml('links.private.yaml',`version: 1
categories:
  - slug: development
    name: Development
    description: Build, documentation, and engineering tools
    color: "#60a5fa"
    order: 10
groups:
  - slug: platform
    name: Platform
    description: Shared infrastructure maintainers
links:
  - slug: ${slug}
    title: ${title}
    url: https://${slug}.example.test
    description: Added through the browser YAML importer
    category: development
    ownerGroup: platform
    icon:
      provider: material
      name: menu_book
      color: "#38bdf8"
    tags: [e2e, imported]
    audience:
      roles: []
      teams: []
`)
    })
    await test.step('See the imported link and merge summary immediately',async()=>{
      await expect(start.linkCard(title)).toBeVisible()
      await expect(page.getByRole('status')).toContainText('Catalog imported: 1 created')
    })
    await test.step('Find the attributed import in administrative history',async()=>{
      const history=await start.openAdministrativeHistory()
      await history.filterFor(slug)
      await expect(history.dialog).toContainText('E2E Administrator')
      await history.close()
    })
  })

  test('restore one selected link without rolling back unrelated links', async ({ page, request }) => {
    const catalog = new CatalogApi(request)
    await catalog.renameSeedLink('stirling-pdf', 'Stirling Changed')
    await catalog.renameSeedLink('pocket-id', 'Pocket Changed')
    const start = new StartPage(page)
    await start.open()

    await test.step('Select only Stirling PDF revision 1 from global history', async () => {
      const history = await start.openAdministrativeHistoryWithKeyboard()
      await history.filterFor('Stirling PDF')
      await history.selectRevision(1)
      await history.restoreSelected(1)
      await history.close()
    })
    await test.step('Confirm Stirling changed back while Pocket ID did not', async () => {
      await start.expectLink('Stirling PDF')
      await start.expectLink('Pocket Changed')
    })
  })

  test('show only permitted features to a non-admin team member', async ({ page }) => {
    await page.route('**/api/v1/bootstrap', route => route.fulfill({
      contentType:'application/json',
      body:JSON.stringify({
        user:{id:99,email:'viewer@example.test',displayName:'Viewer',roles:['reader'],teams:['security'],groups:[],systemAdmin:false,canManageLinks:false},
        branding:{appName:'Start',welcomeText:'',logoUrl:'',accentColor:'#38bdf8',supportUrl:''},
        preferences:defaultPreferences(1),
        categories:[{id:1,slug:'security',name:'Security',description:'',color:'#f59e0b',sortOrder:1}],
        ownerGroups:[{id:1,slug:'platform',name:'Platform',description:'',canCreate:false}],
        links:[{id:99,slug:'team-vault',title:'Team Vault',url:'https://vault.example.test',description:'Security team only',iconProvider:'fontawesome',iconName:'key',iconColor:'',categoryId:1,categorySlug:'security',categoryName:'Security',categoryColor:'#f59e0b',ownerGroupId:1,ownerGroupSlug:'platform',ownerGroupName:'Platform',tags:['secrets'],roles:[],teams:['security'],hidden:false,canManage:false,revision:1}],
      }),
    }))
    const start = new StartPage(page)

    await test.step('Open the launchpad and see the team-authorized link', async () => {
      await start.open()
      await start.expectLink('Team Vault')
    })
    await test.step('Confirm link-management actions are not offered', async () => {
      await expect(page.getByRole('button', { name:/Add link/ })).toHaveCount(0)
      await expect(page.getByRole('button', { name:'History', exact:true })).toHaveCount(0)
      await page.locator('summary[aria-label="Settings"]').click()
      await expect(page.getByRole('button', { name:'Import YAML' })).toHaveCount(0)
      await expect(page.getByRole('button', { name:'Edit' })).toHaveCount(0)
      await expect(page.getByRole('button', { name:'Delete' })).toHaveCount(0)
    })
  })

  test('report service health, useful errors, and sign out safely', async ({ page, request }) => {
    await test.step('Expose healthy readiness and metrics endpoints', async () => {
      await expect((await request.get('/healthz')).json()).resolves.toEqual({status:'ok'})
      await expect((await request.get('/readyz')).json()).resolves.toEqual({status:'ready'})
      expect(await (await request.get('/metrics')).text()).toContain('start_http_requests_total')
    })
    await test.step('Return useful API and browser 404 pages', async () => {
      const missingAPI = await request.get('/api/v1/does-not-exist')
      expect(missingAPI.status()).toBe(404)
      expect((await missingAPI.json()).error.code).toBe('not_found')
      const missingPage = await request.get('/does-not-exist')
      expect(missingPage.status()).toBe(404)
      expect(await missingPage.text()).toContain('Page not found')
    })
    await test.step('Open the account menu and sign out', async () => {
      const start = new StartPage(page)
      await start.open()
      await start.openAccountMenu()
      await start.signOut()
      await expect(page.getByLabel('Account for E2E Administrator')).toBeVisible()
    })
  })
})
