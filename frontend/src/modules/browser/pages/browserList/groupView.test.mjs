import assert from 'node:assert/strict'
import test from 'node:test'

import {
  getDirectChildGroups,
  getGroupPath,
  profileMatchesSelectedGroup,
} from './groupView.ts'

const groups = [
  { groupId: 'account', groupName: '账号', parentId: '', sortOrder: 0, instanceCount: 3 },
  { groupId: 'test-b', groupName: '测试 B', parentId: 'account', sortOrder: 2, instanceCount: 1 },
  { groupId: 'test-a', groupName: '测试 A', parentId: 'account', sortOrder: 1, instanceCount: 2 },
  { groupId: 'deep', groupName: '下一层', parentId: 'test-a', sortOrder: 0, instanceCount: 4 },
]

test('specific groups only match directly assigned profiles', () => {
  assert.equal(profileMatchesSelectedGroup('account', 'account'), true)
  assert.equal(profileMatchesSelectedGroup('test-a', 'account'), false)
  assert.equal(profileMatchesSelectedGroup(undefined, 'account'), false)
})

test('all and ungrouped filters retain their existing meaning', () => {
  assert.equal(profileMatchesSelectedGroup('test-a', ''), true)
  assert.equal(profileMatchesSelectedGroup(undefined, ''), true)
  assert.equal(profileMatchesSelectedGroup(undefined, '__ungrouped__'), true)
  assert.equal(profileMatchesSelectedGroup('account', '__ungrouped__'), false)
})

test('child folder entries include only the next level and respect sort order', () => {
  assert.deepEqual(
    getDirectChildGroups(groups, 'account').map(group => group.groupId),
    ['test-a', 'test-b']
  )
})

test('group paths provide the compact navigation context', () => {
  assert.deepEqual(
    getGroupPath(groups, 'deep').map(group => group.groupName),
    ['账号', '测试 A', '下一层']
  )
})
