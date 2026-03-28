import { Session } from '@supabase/supabase-js';
import { createContext, PropsWithChildren, useEffect, useMemo, useState } from 'react';

import { getAppEnv } from '@/lib/env';
import { supabase } from '@/lib/supabase';

type Credentials = {
  email: string;
  password: string;
};

type AuthResult = {
  error: Error | null;
  needsEmailConfirmation: boolean;
};

type AuthContextValue = {
  authReady: boolean;
  loading: boolean;
  session: Session | null;
  signIn: (credentials: Credentials) => Promise<AuthResult>;
  signOut: () => Promise<void>;
  signUp: (credentials: Credentials) => Promise<AuthResult>;
};

export const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
  const env = getAppEnv();
  const authReady = Boolean(env.supabaseUrl && env.supabaseAnonKey);
  const [loading, setLoading] = useState(authReady);
  const [session, setSession] = useState<Session | null>(null);

  useEffect(() => {
    if (!authReady) {
      setLoading(false);
      setSession(null);
      return;
    }

    let isActive = true;

    supabase.auth.getSession().then(({ data }) => {
      if (!isActive) {
        return;
      }

      setSession(data.session);
      setLoading(false);
    });

    const { data } = supabase.auth.onAuthStateChange((_event, nextSession) => {
      setSession(nextSession);
      setLoading(false);
    });

    return () => {
      isActive = false;
      data.subscription.unsubscribe();
    };
  }, [authReady]);

  const value = useMemo<AuthContextValue>(() => {
    async function signIn(credentials: Credentials): Promise<AuthResult> {
      if (!authReady) {
        return {
          error: new Error('Supabase auth is not configured yet.'),
          needsEmailConfirmation: false,
        };
      }

      const { error } = await supabase.auth.signInWithPassword(credentials);

      return {
        error: error ? new Error(error.message) : null,
        needsEmailConfirmation: false,
      };
    }

    async function signUp(credentials: Credentials): Promise<AuthResult> {
      if (!authReady) {
        return {
          error: new Error('Supabase auth is not configured yet.'),
          needsEmailConfirmation: false,
        };
      }

      const { data, error } = await supabase.auth.signUp(credentials);

      return {
        error: error ? new Error(error.message) : null,
        needsEmailConfirmation: !data.session,
      };
    }

    async function signOut() {
      if (!authReady) {
        return;
      }

      await supabase.auth.signOut();
    }

    return {
      authReady,
      loading,
      session,
      signIn,
      signOut,
      signUp,
    };
  }, [authReady, loading, session]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
