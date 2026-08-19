export type ViewerFollowState = 'none' | 'pending' | 'following';
export type FollowButtonAction = 'follow' | 'unfollow';

export type FollowButtonState = {
	label: string;
	action: FollowButtonAction;
};

export function getFollowButtonState(input: {
	viewerFollowState: ViewerFollowState;
	restrictFollow: boolean;
}): FollowButtonState {
	if (input.viewerFollowState === 'pending') {
		return { label: 'Requested', action: 'unfollow' };
	}
	if (input.viewerFollowState === 'following') {
		return { label: 'Following', action: 'unfollow' };
	}
	return {
		label: input.restrictFollow ? 'Request to follow' : 'Follow',
		action: 'follow'
	};
}
