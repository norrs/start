import { expect, test, type Locator, type Page } from '@playwright/test'

export type NewLink = {
  title:string; slug:string; url:string; description:string
  iconProvider:'devicon'|'fontawesome'|'material'; iconName:string; tags:string
}

export class StartPage {
  readonly search:Locator
  constructor(readonly page:Page) { this.search = page.getByRole('searchbox', { name:'Search links' }) }
  private async pressKey(key:string, title:string) {
    await test.step(title, () => this.page.keyboard.press(key))
  }
  async open() { await this.page.goto('/') }
  linkCard(title:string) { return this.page.locator('.link-card').filter({ hasText:title }) }

  async expectAdministratorHome() {
    await expect(this.page.getByRole('link', { name:'Start home' })).toBeVisible()
    await expect(this.page.getByLabel('Account for E2E Administrator')).toContainText('Welcome E2E Administrator')
    await expect(this.page.getByRole('heading', { name:'Development' })).toBeVisible()
    await expect(this.page.getByRole('heading', { name:'Security' })).toBeVisible()
  }
  async searchFor(query:string) { await this.search.fill(query) }
  async clearSearch() { await this.search.fill('') }
  async expectSingleResult(title:string) {
    await expect(this.page.locator('.link-card')).toHaveCount(1)
    await expect(this.page.getByText(title, { exact:true })).toBeVisible()
  }
  async expectResult(title:string) { await expect(this.page.getByText(title, { exact:true })).toBeVisible() }
  async focusSearchWithKeyboard() {
    await this.pressKey('Control+k', 'Press Ctrl+K to focus search')
    await expect(this.search).toBeFocused()
  }
  async clearSearchWithKeyboard() {
    await this.pressKey('Escape', 'Press Escape to clear search')
    await expect(this.search).toHaveValue('')
  }
  async openKeyboardHelp() {
    await this.search.blur()
    await this.pressKey('?', 'Press ? to open keyboard help')
    await expect(this.page.getByRole('dialog', { name:'Keyboard shortcuts' })).toBeVisible()
  }
  async closeKeyboardHelp() {
    await this.pressKey('Escape', 'Press Escape to close keyboard help')
    await expect(this.page.getByRole('dialog', { name:'Keyboard shortcuts' })).toBeHidden()
  }
  async selectNextLinkWithKeyboard(title:string) {
    await this.pressKey('j', 'Press J to select the next link')
    await expect(this.page.locator('.link-card.active')).toContainText(title)
  }
  async toggleViewWithKeyboard(expectedView:'All links'|'Categories') {
    await this.pressKey('v', `Press V to switch to ${expectedView}`)
    await expect(this.page.getByRole('button', { name:expectedView })).toHaveClass(/selected/)
  }
  async openOnlySearchResult() {
    await test.step('Press Enter to open the only search result', () => this.search.press('Enter'))
  }
  async openDisplaySettings() {
    await this.page.locator('summary[aria-label="Settings"]').click()
    await this.page.getByRole('button', { name:'Display settings' }).click()
    return new DisplaySettings(this.page)
  }
  async hideSelectedLinkWithKeyboard() {
    await this.pressKey('h', 'Press H to hide the selected link')
    await expect(this.page.getByRole('status')).toHaveText('Link hidden')
  }
  async showHiddenLink(title:string) {
    await this.linkCard(title).getByRole('button', { name:`Show ${title}` }).click()
    await expect(this.page.getByRole('status')).toHaveText('Link shown')
  }
  async openNewLinkEditorWithKeyboard() {
    await expect(this.page.getByRole('button', { name:/Add link/ })).toBeVisible()
    await this.pressKey('n', 'Press N to create a new link')
    return new LinkEditor(this.page, 'Add link')
  }
  async openLinkEditor(title:string) {
    await this.linkCard(title).getByRole('button', { name:'Edit' }).click()
    return new LinkEditor(this.page, `Edit ${title}`)
  }
  async openLinkHistory(title:string) {
    await this.linkCard(title).getByRole('button', { name:'History' }).click()
    return new LinkHistory(this.page, title)
  }
  async deleteLink(title:string) {
    this.page.once('dialog', confirmation => confirmation.accept())
    await this.linkCard(title).getByRole('button', { name:'Delete' }).click()
    await expect(this.page.getByRole('status')).toHaveText('Link deleted')
    await expect(this.linkCard(title)).toHaveCount(0)
  }
  async openAdministrativeHistoryWithKeyboard() {
    await this.pressKey('g', 'Press G to open link change history')
    return new AdministrativeHistory(this.page)
  }
  async openAdministrativeHistory() {
    await this.page.getByRole('navigation', { name:'Account and settings' }).getByRole('button', { name:'History' }).click()
    return new AdministrativeHistory(this.page)
  }
  async openCatalogImportWithKeyboard() {
    await this.pressKey('i', 'Press I to open the catalog importer')
    return new CatalogImport(this.page)
  }
  async openCatalogImportFromSettings() {
    await this.page.locator('summary[aria-label="Settings"]').click()
    await this.page.getByRole('button', { name:'Import YAML' }).click()
    return new CatalogImport(this.page)
  }
  async expectLink(title:string) { await expect(this.page.getByText(title, { exact:true })).toBeVisible() }
  async openAccountMenu() { await this.page.getByLabel('Account for E2E Administrator').click() }
  async signOut() {
    const request = this.page.waitForRequest(candidate => candidate.url().endsWith('/auth/logout') && candidate.method() === 'POST')
    await this.page.getByRole('button', { name:'Sign out' }).click()
    await request
  }
}

export class DisplaySettings {
  readonly dialog:Locator
  constructor(private readonly page:Page) { this.dialog = page.getByRole('dialog', { name:'Display settings' }) }
  private async save(action:()=>Promise<unknown>) {
    await Promise.all([
      this.page.waitForResponse(response => response.url().endsWith('/api/v1/me/preferences') && response.request().method() === 'PUT'),
      action(),
    ])
  }
  async chooseTheme(theme:'dark'|'light'|'hacker'|'1984') { await this.save(() => this.dialog.getByLabel('Theme').selectOption(theme)) }
  async showDescriptions(show:boolean) { await this.save(() => this.dialog.getByLabel('Show descriptions').setChecked(show)) }
  async showCategoryHeadings(show:boolean) { await this.save(() => this.dialog.getByLabel('Show category headings').setChecked(show)) }
  async showIcons(show:boolean) { await this.save(() => this.dialog.getByLabel('Show icons').setChecked(show)) }
  async showHiddenLinks(show:boolean) { await this.save(() => this.dialog.getByLabel('Show links I have hidden').setChecked(show)) }
  async close() { await this.page.keyboard.press('Escape') }
}

export class LinkEditor {
  readonly dialog:Locator
  constructor(private readonly page:Page, name:string) { this.dialog = page.getByRole('dialog', { name }) }
  async fillLink(link:NewLink) {
    await this.dialog.getByLabel('Title').fill(link.title)
    await this.dialog.getByLabel('Slug').fill(link.slug)
    await this.dialog.getByLabel('URL').fill(link.url)
    await this.dialog.getByLabel('Description').fill(link.description)
    await this.dialog.getByLabel('Icon provider').selectOption(link.iconProvider)
    await this.dialog.getByLabel('Icon name').fill(link.iconName)
    await this.dialog.getByLabel('Tags').fill(link.tags)
  }
  async expectMaterialIconPreview(iconName:string) { await expect(this.dialog.locator('.icon-preview .material-icons')).toHaveText(iconName) }
  async enterAudienceRules(roles:string, teams:string) {
    await this.dialog.getByLabel('Roles').fill(roles)
    await this.dialog.getByLabel('Teams').fill(teams)
    await expect(this.dialog.getByLabel('Roles')).toHaveValue(roles)
    await expect(this.dialog.getByLabel('Teams')).toHaveValue(teams)
  }
  async clearAudienceRules() {
    await this.dialog.getByLabel('Roles').fill('')
    await this.dialog.getByLabel('Teams').fill('')
  }
  async changeTitle(title:string) { await this.dialog.getByLabel('Title').fill(title) }
  async changeIcon(iconName:string) { await this.dialog.getByLabel('Icon name').fill(iconName) }
  async save() {
    await this.dialog.getByRole('button', { name:'Save link' }).click()
    await expect(this.page.getByRole('status')).toHaveText('Link saved')
  }
}

export class LinkHistory {
  readonly dialog:Locator
  constructor(private readonly page:Page, title:string) { this.dialog = page.getByRole('dialog', { name:`${title} history` }) }
  async expectRevision(revision:number) { await expect(this.dialog.getByText(`Revision ${revision}`, { exact:true })).toBeVisible() }
  async restoreRevision(revision:number) {
    this.page.once('dialog', confirmation => confirmation.accept())
    await this.dialog.locator('li').filter({ hasText:`Revision ${revision}` }).getByRole('button', { name:'Restore' }).click()
  }
}

export class AdministrativeHistory {
  readonly dialog:Locator
  constructor(private readonly page:Page) { this.dialog = page.getByRole('dialog', { name:'Link change history' }) }
  async filterFor(query:string) { await this.dialog.getByLabel('Filter history').fill(query) }
  async selectRevision(revision:number) { await this.dialog.locator('li').filter({ hasText:`revision ${revision}` }).getByRole('checkbox').check() }
  async restoreSelected(count:number) {
    this.page.once('dialog', confirmation => confirmation.accept())
    await this.dialog.getByRole('button', { name:`Restore ${count}` }).click()
    await expect(this.page.getByRole('status')).toHaveText(`${count} link${count === 1 ? '' : 's'} restored`)
  }
  async close() { await this.page.keyboard.press('Escape') }
}

export class CatalogImport {
  readonly dialog:Locator
  constructor(private readonly page:Page) { this.dialog = page.getByRole('dialog', { name:'Import link catalog' }) }
  async importYaml(name:string, contents:string) {
    await this.dialog.getByLabel('Catalog YAML').setInputFiles({name,mimeType:'application/yaml',buffer:Buffer.from(contents)})
    await expect(this.dialog.getByText(name, { exact:true })).toBeVisible()
    await expect(this.dialog.locator('pre')).toContainText('version: 1')
    await this.dialog.getByLabel(/I understand that matching catalog records/).check()
    await this.dialog.getByRole('button', { name:'Import catalog' }).click()
  }
}
