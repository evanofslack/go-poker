import React from 'react'
import { AppProps } from 'next/app'
import { SocketProvider } from '../providers/WebSocket'
import { AppStateProvider } from '../providers/AppStore'

import '../styles/index.css'

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <AppStateProvider>
      <SocketProvider>
        <Component {...pageProps} />
      </SocketProvider>
    </AppStateProvider>
  )
}

export default MyApp