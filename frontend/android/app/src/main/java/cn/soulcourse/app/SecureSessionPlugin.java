package cn.soulcourse.app;

import android.content.Context;
import android.content.SharedPreferences;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.util.Base64;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.nio.charset.StandardCharsets;
import java.security.KeyStore;
import java.security.SecureRandom;

import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.spec.GCMParameterSpec;
import javax.crypto.spec.SecretKeySpec;

@CapacitorPlugin(name = "SecureSession")
public class SecureSessionPlugin extends Plugin {
    private static final String KEY_ALIAS = "cn.soulcourse.app.session.v1";
    private static final String PREFS = "secure_session";
    private static final String VALUE = "encrypted_value";
    private static final String IV = "iv";

    @PluginMethod
    public void get(PluginCall call) {
        try {
            SharedPreferences prefs = preferences();
            String encrypted = prefs.getString(VALUE, null);
            String encodedIv = prefs.getString(IV, null);
            JSObject result = new JSObject();
            if (encrypted != null && encodedIv != null) {
                result.put("value", decrypt(encrypted, encodedIv));
            }
            call.resolve(result);
        } catch (Exception error) {
            call.reject("secure session is unavailable", error);
        }
    }

    @PluginMethod
    public void set(PluginCall call) {
        String value = call.getString("value", "").trim();
        if (value.isEmpty()) {
            call.reject("value is required");
            return;
        }
        try {
            byte[] iv = new byte[12];
            new SecureRandom().nextBytes(iv);
            byte[] encrypted = cipher(Cipher.ENCRYPT_MODE, iv).doFinal(value.getBytes(StandardCharsets.UTF_8));
            preferences().edit()
                    .putString(VALUE, Base64.encodeToString(encrypted, Base64.NO_WRAP))
                    .putString(IV, Base64.encodeToString(iv, Base64.NO_WRAP))
                    .apply();
            call.resolve();
        } catch (Exception error) {
            call.reject("could not store secure session", error);
        }
    }

    @PluginMethod
    public void remove(PluginCall call) {
        preferences().edit().clear().apply();
        call.resolve();
    }

    private SharedPreferences preferences() {
        return getContext().getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }

    private String decrypt(String encryptedValue, String encodedIv) throws Exception {
        byte[] encrypted = Base64.decode(encryptedValue, Base64.NO_WRAP);
        byte[] iv = Base64.decode(encodedIv, Base64.NO_WRAP);
        byte[] plain = cipher(Cipher.DECRYPT_MODE, iv).doFinal(encrypted);
        return new String(plain, StandardCharsets.UTF_8);
    }

    private Cipher cipher(int mode, byte[] iv) throws Exception {
        Cipher cipher = Cipher.getInstance("AES/GCM/NoPadding");
        cipher.init(mode, key(), new GCMParameterSpec(128, iv));
        return cipher;
    }

    private SecretKey key() throws Exception {
        KeyStore keyStore = KeyStore.getInstance("AndroidKeyStore");
        keyStore.load(null);
        if (!keyStore.containsAlias(KEY_ALIAS)) {
            KeyGenerator generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore");
            generator.init(new KeyGenParameterSpec.Builder(
                    KEY_ALIAS,
                    KeyProperties.PURPOSE_ENCRYPT | KeyProperties.PURPOSE_DECRYPT)
                    .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                    .setRandomizedEncryptionRequired(true)
                    .build());
            generator.generateKey();
        }
        return ((javax.crypto.SecretKey) keyStore.getKey(KEY_ALIAS, null));
    }
}
