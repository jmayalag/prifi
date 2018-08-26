package ch.epfl.prifiproxy.activities;

import android.arch.lifecycle.ViewModelProviders;
import android.content.Context;
import android.content.Intent;
import android.os.Bundle;
import android.support.design.widget.TextInputEditText;
import android.support.design.widget.TextInputLayout;
import android.support.v7.app.AlertDialog;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;
import android.text.TextUtils;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;

import java.util.Objects;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.ui.Mode;
import ch.epfl.prifiproxy.utils.NetworkHelper;
import ch.epfl.prifiproxy.viewmodel.ConfigurationAddEdditViewModel;

public class ConfigurationAddEditActivity extends AppCompatActivity {
    private static final String TAG = "CONF_ADD_EDIT_ACTIVITY";
    protected static final String EXTRA_CONFIGURATION_ID = "configurationId";
    protected static final String EXTRA_GROUP_ID = "groupId";
    private static final String EXTRA_MODE = "mode";

    private int mode;
    private int menuId;

    private Configuration configuration;
    private Toolbar toolbar;

    private TextInputLayout nameInputLayout;
    private TextInputEditText nameInput;
    private TextInputLayout hostInputLayout;
    private TextInputEditText hostInput;
    private TextInputLayout relayPortInputLayout;
    private TextInputEditText relayPortInput;
    private TextInputLayout socksPortInputLayout;
    private TextInputEditText socksPortInput;

    private ConfigurationAddEdditViewModel viewModel;
    private AlertDialog.Builder deleteDialogBuilder;

    private static Intent getIntent(Context packageContext) {
        return new Intent(packageContext, ConfigurationAddEditActivity.class);
    }

    public static Intent intentAdd(Context packageContext, ConfigurationGroup group) {
        Intent intent = getIntent(packageContext);
        intent.putExtra(EXTRA_MODE, Mode.ADD);
        intent.putExtra(EXTRA_GROUP_ID, group.getId());
        intent.putExtra(EXTRA_CONFIGURATION_ID, 0);
        return intent;
    }

    public static Intent intentEdit(Context packageContext, Configuration configuration) {
        Intent intent = getIntent(packageContext);
        intent.putExtra(EXTRA_MODE, Mode.EDIT);
        intent.putExtra(EXTRA_GROUP_ID, configuration.getGroupId());
        intent.putExtra(EXTRA_CONFIGURATION_ID, configuration.getId());
        return intent;
    }

    public static Intent intentDetail(Context packageContext, Configuration configuration) {
        Intent intent = getIntent(packageContext);
        intent.putExtra(EXTRA_MODE, Mode.DETAIL);
        intent.putExtra(EXTRA_GROUP_ID, configuration.getGroupId());
        intent.putExtra(EXTRA_CONFIGURATION_ID, configuration.getId());
        return intent;
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_configuration_add_edit);
        toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        nameInputLayout = findViewById(R.id.nameInputLayout);
        nameInput = findViewById(R.id.nameInput);

        hostInputLayout = findViewById(R.id.hostInputLayout);
        hostInput = findViewById(R.id.hostInput);

        relayPortInputLayout = findViewById(R.id.relayPortInputLayout);
        relayPortInput = findViewById(R.id.relayPortInput);

        socksPortInputLayout = findViewById(R.id.socksPortInputLayout);
        socksPortInput = findViewById(R.id.socksPortInput);


        viewModel = ViewModelProviders.of(this).get(ConfigurationAddEdditViewModel.class);

        Bundle bundle = getIntent().getExtras();

        if (bundle == null) {
            // Default is adding mode
            mode = Mode.ADD;
        } else {
            mode = bundle.getInt(EXTRA_MODE, Mode.ADD);
            int configurationId = bundle.getInt(EXTRA_CONFIGURATION_ID, -1);
            int groupId = bundle.getInt(EXTRA_GROUP_ID, -1);

            if (groupId == -1) {
                Log.w(TAG, "Invalid state, group id must be provided");
                finish();
            }

            if (configurationId != -1) {
                viewModel.init(groupId, configurationId);
                viewModel.getConfiguration().observe(this, this::bindView);
            }
        }

        if (mode == Mode.ADD || mode == Mode.EDIT) {
            menuId = R.menu.menu_configuration_add_edit;
        } else {
            menuId = R.menu.menu_configuration_detail;
            nameInput.setKeyListener(null);
            hostInput.setKeyListener(null);
            relayPortInput.setKeyListener(null);
            socksPortInput.setKeyListener(null);
            nameInput.setFocusable(false);
            hostInput.setFocusable(false);
            relayPortInput.setFocusable(false);
            socksPortInput.setFocusable(false);
        }

        bindView(configuration);

        deleteDialogBuilder = new AlertDialog.Builder(this)
                .setTitle(R.string.dialog_delete_configuration_title)
                .setMessage(R.string.dialog_delete_configuration_msg)
                .setCancelable(true)
                .setNegativeButton(android.R.string.no, null)
                .setPositiveButton(android.R.string.yes, (dialog, which) -> deleteConfiguration());
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(menuId, menu);

        return true;
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        int id = item.getItemId();

        switch (id) {
            case android.R.id.home:
                finish(); //TODO: Fix up navigation
                return true;
            case R.id.save_configuration:
                saveConfiguration();
                return true;
            case R.id.edit_configuration:
                editConfiguration();
                return true;
            case R.id.delete_configuration:
                deleteDialogBuilder.show();
                return true;
        }

        return super.onOptionsItemSelected(item);
    }

    private void bindView(Configuration configuration) {
        if (configuration == null) {
            socksPortInput.setText(R.string.default_socks_port);
            return;
        }

        this.configuration = configuration;
        nameInput.setText(configuration.getName());
        hostInput.setText(configuration.getHost());
        relayPortInput.setText(String.valueOf(configuration.getRelayPort()));
        socksPortInput.setText(String.valueOf(configuration.getSocksPort()));
        Objects.requireNonNull(getSupportActionBar()).setTitle(configuration.getName());
    }

    private void saveConfiguration() {
        Configuration configuration = this.configuration;
        Configuration validated = validate();

        if (validated == null) {
            return;
        }

        if (configuration == null) {
            // Add a new configuration
            configuration = new Configuration();
            configuration.setGroupId(viewModel.getGroupId());
        }
        configuration.setName(validated.getName());
        configuration.setHost(validated.getHost());
        configuration.setRelayPort(validated.getRelayPort());
        configuration.setSocksPort(validated.getSocksPort());

        viewModel.insertOrUpdate(configuration);
        finish();
    }

    private Configuration validate() {
        boolean isValid = true;
        String name = nameInput.getText().toString();
        String host = hostInput.getText().toString();
        String relayPort = relayPortInput.getText().toString();
        String socksPort = socksPortInput.getText().toString();

        resetErrors();

        if (TextUtils.isEmpty(name)) {
            nameInputLayout.setError(getText(R.string.required));
            isValid = false;
        }
        if (TextUtils.isEmpty(host)) {
            hostInputLayout.setError(getText(R.string.required));
            isValid = false;
        }
        if (TextUtils.isEmpty(relayPort)) {
            relayPortInput.setError(getText(R.string.required));
            isValid = false;
        }
        if (TextUtils.isEmpty(socksPort)) {
            socksPortInputLayout.setError(getText(R.string.required));
            isValid = false;
        }

        if (!isValid) {
            return null;
        }

        if (!NetworkHelper.isValidIpv4Address(host)) {
            hostInputLayout.setError(getString(R.string.msg_invalid_ip_address));
            isValid = false;
        }

        if (!NetworkHelper.isValidPort(relayPort)) {
            relayPortInputLayout.setError(getString(R.string.msg_invalid_port));
            isValid = false;
        }
        if (!NetworkHelper.isValidPort(socksPort)) {
            socksPortInputLayout.setError(getString(R.string.msg_invalid_port));
            isValid = false;
        }

        if (!isValid) {
            return null;
        }
        Configuration configuration = new Configuration();
        configuration.setName(name);
        configuration.setHost(host);
        configuration.setRelayPort(Integer.parseInt(relayPort));
        configuration.setSocksPort(Integer.parseInt(socksPort));
        return configuration;
    }

    private void resetErrors() {
        nameInputLayout.setError(null);
        hostInputLayout.setError(null);
        relayPortInputLayout.setError(null);
        socksPortInputLayout.setError(null);
    }

    private void editConfiguration() {
        if (configuration == null) {
            Log.e(TAG, "Illegal state. Can't edit null configuration");
            finish();
        }
        startActivity(intentEdit(this, configuration));
    }

    private void deleteConfiguration() {
        viewModel.delete(configuration);
        finish();
    }
}
