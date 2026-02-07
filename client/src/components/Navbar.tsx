'use client';

import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/authStore';

export default function Navbar() {
    const router = useRouter();
    const { user, logout } = useAuthStore();

    const handleLogout = () => {
        logout();
        router.push('/login');
    };

    return (
        <nav className="bg-white border-b border-gray-200">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                <div className="flex justify-between items-center h-16">
                    <div className="flex items-center space-x-8">
                        <h1
                            className="text-2xl font-bold text-[#1479b0] cursor-pointer"
                            onClick={() => router.push('/dashboard')}
                        >
                            Docs Editor
                        </h1>
                    </div>

                    <div className="flex items-center space-x-4">
                        {user && (
                            <>
                                <span className="text-sm text-gray-600">
                                    {user.name}
                                </span>
                                <button
                                    onClick={handleLogout}
                                    className="px-4 py-2 text-sm text-gray-700 hover:text-[#1479b0] transition"
                                >
                                    Logout
                                </button>
                            </>
                        )}
                    </div>
                </div>
            </div>
        </nav>
    );
}