package ch.epfl.prifiproxy.repository;

import android.app.Application;
import android.arch.lifecycle.LiveData;
import android.os.AsyncTask;

import java.util.List;

import ch.epfl.prifiproxy.persistence.AppDatabase;
import ch.epfl.prifiproxy.persistence.dao.ConfigurationDao;
import ch.epfl.prifiproxy.persistence.dao.ConfigurationGroupDao;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

public class ConfigurationRepository {
    private ConfigurationDao configurationDao;

    public ConfigurationRepository(Application application) {
        AppDatabase db = AppDatabase.getDatabase(application);
        configurationDao = db.configurationDao();
    }

    public LiveData<List<Configuration>> getConfigurations(int groupId) {
        return configurationDao.getForGroup(groupId);
    }

    public void insert(Configuration configuration) {
        new InsertAsyncTask(configurationDao).execute(configuration);
    }

    public void update(Configuration configuration) {
        new UpdateAsyncTask(configurationDao).execute(configuration);
    }

    public void update(List<Configuration> configurations) {
        new UpdateAsyncTask(configurationDao)
                .execute(configurations.toArray(new Configuration[configurations.size()]));
    }

    public void delete(Configuration configuration) {
        new DeleteAsyncTask(configurationDao).execute(configuration);
    }

    private static class InsertAsyncTask extends AsyncTask<Configuration, Void, Void> {
        private final ConfigurationDao dao;

        InsertAsyncTask(ConfigurationDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(final Configuration... configurations) {
            dao.insert(configurations);
            return null;
        }
    }

    private static class UpdateAsyncTask extends AsyncTask<Configuration, Void, Void> {
        private final ConfigurationDao dao;

        UpdateAsyncTask(ConfigurationDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(Configuration... configurations) {
            dao.update(configurations);
            return null;
        }
    }

    private static class DeleteAsyncTask extends AsyncTask<Configuration, Void, Void> {
        private final ConfigurationDao dao;

        DeleteAsyncTask(ConfigurationDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(Configuration... configurations) {
            dao.delete(configurations);
            return null;
        }
    }
}
