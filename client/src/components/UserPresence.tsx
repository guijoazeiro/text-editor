import { UserPresence as UserPresenceType } from '@/lib/websocket';

interface Props {
    users: UserPresenceType[];
    currentUserId: string;
}

export default function UserPresence({ users, currentUserId }: Props) {
    const otherUsers = users.filter((u) => u.user_id !== currentUserId);

    return (
        <div className="flex items-center space-x-2">
            <span className="text-sm text-gray-600">Online:</span>
            <div className="flex -space-x-2">
                {otherUsers.slice(0, 5).map((user) => (
                    <div
                        key={user.user_id}
                        className="w-8 h-8 rounded-full border-2 border-white flex items-center justify-center text-xs font-medium text-white"
                        style={{ backgroundColor: user.color }}
                        title={user.user_name}
                    >
                        {user.user_name.charAt(0).toUpperCase()}
                    </div>
                ))}
                {otherUsers.length > 5 && (
                    <div className="w-8 h-8 rounded-full border-2 border-white bg-gray-400 flex items-center justify-center text-xs font-medium text-white">
                        +{otherUsers.length - 5}
                    </div>
                )}
            </div>
            {otherUsers.length === 0 && (
                <span className="text-sm text-gray-400">Just you</span>
            )}
        </div>
    );
}