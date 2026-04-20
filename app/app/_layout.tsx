import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import 'react-native-reanimated';

import { AuthProvider } from '@/providers/AuthProvider';
export { ErrorBoundary } from 'expo-router';

export default function RootLayout() {
  return (
    <AuthProvider>
      <StatusBar style="dark" />
      <Stack
        screenOptions={{
          headerShadowVisible: false,
          headerStyle: {
            backgroundColor: '#f3ead9',
          },
          headerTitleStyle: {
            color: '#1f3527',
            fontWeight: '600',
          },
          contentStyle: {
            backgroundColor: '#f7f2e8',
          },
        }}>
        <Stack.Screen name="index" options={{ title: 'FreshCycle' }} />
        <Stack.Screen name="auth" options={{ title: 'Sign In or Sign Up' }} />
        <Stack.Screen name="add-garment" options={{ title: 'Add Garment' }} />
        <Stack.Screen name="plan-load" options={{ title: 'Plan A Load' }} />
        <Stack.Screen name="load-detail" options={{ title: 'Load Details' }} />
      </Stack>
    </AuthProvider>
  );
}
