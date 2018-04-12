package ch.epfl.prifiproxy.utils;

import android.util.Log;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.net.SocketAddress;

public class NetworkStatusHelper {

    private static final String TAG = NetworkStatusHelper.class.getName();

    public static boolean isHostReachable(String serverAddress, int serverTcpPort, int timeout){
        boolean connected = false;
        Socket socket;

        try {
            socket = new Socket();
            SocketAddress socketAddress = new InetSocketAddress(serverAddress, serverTcpPort);
            socket.connect(socketAddress, timeout);
            if (socket.isConnected()) {
                connected = true;
                socket.close();
            }
        } catch (IOException e) {
            Log.i(TAG, "Cannot connect to the host " + serverAddress + ":" + String.valueOf(serverTcpPort));
        }

        return connected;
    }

}
