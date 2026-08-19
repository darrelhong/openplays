import { describe, expect, it } from 'vitest';
import { getFollowButtonState } from './profile-follow';

describe('getFollowButtonState', () => {
	it.each([
		{ state: 'none' as const, restrictFollow: false, label: 'Follow', action: 'follow' as const },
		{
			state: 'none' as const,
			restrictFollow: true,
			label: 'Request to follow',
			action: 'follow' as const
		},
		{
			state: 'pending' as const,
			restrictFollow: false,
			label: 'Requested',
			action: 'unfollow' as const
		},
		{
			state: 'pending' as const,
			restrictFollow: true,
			label: 'Requested',
			action: 'unfollow' as const
		},
		{
			state: 'following' as const,
			restrictFollow: false,
			label: 'Following',
			action: 'unfollow' as const
		},
		{
			state: 'following' as const,
			restrictFollow: true,
			label: 'Following',
			action: 'unfollow' as const
		}
	])('$state with restrict_follow=$restrictFollow maps to $label / $action', (testCase) => {
		expect(
			getFollowButtonState({
				viewerFollowState: testCase.state,
				restrictFollow: testCase.restrictFollow
			})
		).toEqual({ label: testCase.label, action: testCase.action });
	});
});
