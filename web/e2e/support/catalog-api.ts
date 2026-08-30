import { expect, type APIRequestContext } from '@playwright/test'

export type Preferences = {
  theme: 'dark' | 'light' | 'hacker' | '1984'
  viewMode: 'categorized' | 'flat'
  showDescriptions: boolean
  showCategories: boolean
  showIcons: boolean
  coloredIcons: boolean
  showHidden: boolean
  openNewTab: boolean
  hotkeysEnabled: boolean
  revision: number
}

type Link = { id:number; slug:string; title:string; revision:number }
export type Bootstrap = { preferences:Preferences; links:Link[] }

export const defaultPreferences = (revision:number):Preferences => ({
  theme: 'dark', viewMode: 'categorized', showDescriptions: true,
  showCategories: true, showIcons: true, coloredIcons: true,
  showHidden: false, openNewTab: false, hotkeysEnabled: true, revision,
})

export class CatalogApi {
  constructor(private readonly request:APIRequestContext) {}

  async bootstrap():Promise<Bootstrap> {
    const response = await this.request.get('/api/v1/bootstrap')
    expect(response.ok()).toBeTruthy()
    return response.json()
  }

  async resetUserExperience() {
    const state = await this.bootstrap()
    const saved = await this.request.put('/api/v1/me/preferences', { data:defaultPreferences(state.preferences.revision) })
    expect(saved.ok()).toBeTruthy()
    for (const link of state.links) {
      expect((await this.request.delete(`/api/v1/me/hidden-links/${link.id}`)).ok()).toBeTruthy()
    }
  }

  async removeTestChanges() {
    const state = await this.bootstrap()
    const seedTitles:Record<string,string> = {'stirling-pdf':'Stirling PDF','pocket-id':'Pocket ID'}
    for (const link of state.links) {
      if (link.slug.startsWith('e2e-toolbox-') || link.slug.startsWith('e2e-import-')) {
        expect((await this.request.delete(`/api/v1/links/${link.id}?revision=${link.revision}`)).ok()).toBeTruthy()
      } else if (seedTitles[link.slug] && link.title !== seedTitles[link.slug]) {
        expect((await this.request.post(`/api/v1/links/${link.id}/restore/1`)).ok()).toBeTruthy()
      }
    }
  }

  async renameSeedLink(slug:string, title:string) {
    const response = await this.request.get('/api/v1/bootstrap')
    expect(response.ok()).toBeTruthy()
    const state = await response.json()
    const link = state.links.find((candidate:any) => candidate.slug === slug)
    expect(link, `seed link ${slug}`).toBeTruthy()
    const input = {
      slug:link.slug,title,url:link.url,description:link.description,iconProvider:link.iconProvider,
      iconName:link.iconName,iconColor:link.iconColor,categoryId:link.categoryId,ownerGroupId:link.ownerGroupId,
      tags:link.tags,roles:link.roles,teams:link.teams,enabled:true,revision:link.revision,
    }
    expect((await this.request.put(`/api/v1/links/${link.id}`, { data:input })).ok()).toBeTruthy()
  }
}
